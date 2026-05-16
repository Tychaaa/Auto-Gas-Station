// Mock онлайн-кассы PayOnline-01-ФА.
//
// Реализует TCP-сервер, говорящий по протоколу ККТ A.2.0
// "стандартный нижний уровень" (ENQ/ACK/NAK + STX/LEN/CMD/DATA/LRC),
// которого достаточно для текущей логики server/internal/adapter/fiscal.
//
// Поддерживаемые команды: 0x10 (ShortStatus), 0x17 (PrintString),
// 0x41 (CloseShiftZ), 0xE0 (OpenShift), 0xFF37 (ReportCalcStart),
// 0xFF38 (ReportCalcForm), 0xFF40 (ShiftParams), 0xFF45 (CloseReceiptV2),
// 0xFF46 (OperationV2). Любую другую команду мок отклоняет кодом 0x42.
package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

// Служебные байты нижнего уровня (см. документ "Протокол ККТ A.2.0").
const (
	enq byte = 0x05
	stx byte = 0x02
	ack byte = 0x06
	nak byte = 0x15
)

// Поддерживаемые коды команд.
const (
	cmdShortStatus    uint16 = 0x0010
	cmdPrintString    uint16 = 0x0017
	cmdCloseShiftZ    uint16 = 0x0041
	cmdOpenShift      uint16 = 0x00E0
	cmdReportCalcStart uint16 = 0xFF37
	cmdReportCalcForm  uint16 = 0xFF38
	cmdShiftParams    uint16 = 0xFF40
	cmdCloseReceipt   uint16 = 0xFF45
	cmdOperationV2    uint16 = 0xFF46
)

type scenario string

const (
	scenarioSuccess        scenario = "success"
	scenarioShiftClosed    scenario = "shift_closed"
	scenarioShiftExpired   scenario = "shift_expired"
	scenarioOperationError scenario = "operation_error"
	scenarioCloseError     scenario = "close_error"
)

type mockConfig struct {
	host          string
	port          int
	scenario      scenario
	initShiftState byte
	initShiftNum  uint16
	initReceiptNum uint16
	dumpHex       bool
}

// shiftState хранит мутабельное состояние смены, защищённое мьютексом.
type shiftState struct {
	mu            sync.Mutex
	state         byte   // 0=closed, 1=open, 2=expired
	shiftNumber   uint16
	receiptNumber uint16
}

type kktState struct {
	cfg       mockConfig
	shift     shiftState
	fdCounter uint32
	fsCounter uint32
}

func main() {
	loadEnv()
	cfg := readConfig()

	state := &kktState{
		cfg: cfg,
		shift: shiftState{
			state:         cfg.initShiftState,
			shiftNumber:   cfg.initShiftNum,
			receiptNumber: cfg.initReceiptNum,
		},
		fdCounter: 1000,
		fsCounter: 0xCAFE0000,
	}

	addr := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("kkt mock: listen %s failed: %v", addr, err)
	}
	defer ln.Close()

	log.Printf("kkt mock: listening on %s", addr)
	log.Printf("kkt mock: scenario=%s shift=%s shift_number=%d receipt_number=%d dump_hex=%v",
		cfg.scenario, shiftStateName(cfg.initShiftState), cfg.initShiftNum, cfg.initReceiptNum, cfg.dumpHex)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("kkt mock: accept failed: %v", err)
			continue
		}
		go handleConn(conn, state)
	}
}

func handleConn(conn net.Conn, st *kktState) {
	defer conn.Close()
	log.Printf("kkt mock: client connected from %s", conn.RemoteAddr())

	rd := bufio.NewReader(conn)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Minute)); err != nil {
			log.Printf("kkt mock: set read deadline: %v", err)
			return
		}

		b, err := rd.ReadByte()
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Printf("kkt mock: read first byte: %v", err)
			}
			log.Printf("kkt mock: client %s disconnected", conn.RemoteAddr())
			return
		}

		if st.cfg.dumpHex {
			log.Printf("kkt mock: rx_byte 0x%02X (%s)", b, byteName(b))
		}

		switch b {
		case enq:
			if err := writeByte(conn, nak); err != nil {
				log.Printf("kkt mock: write NAK on ENQ: %v", err)
				return
			}
			if st.cfg.dumpHex {
				log.Printf("kkt mock: tx_byte 0x15 (NAK)")
			}

		case stx:
			if err := handleFrame(rd, conn, st); err != nil {
				log.Printf("kkt mock: handle frame: %v", err)
				return
			}

		case ack, nak:
			if st.cfg.dumpHex {
				log.Printf("kkt mock: stray byte 0x%02X (%s) outside frame, ignored", b, byteName(b))
			}

		default:
			log.Printf("kkt mock: unexpected byte 0x%02X, ignoring", b)
		}
	}
}

func handleFrame(rd *bufio.Reader, conn net.Conn, st *kktState) error {
	lenByte, err := rd.ReadByte()
	if err != nil {
		return fmt.Errorf("read LEN: %w", err)
	}
	if lenByte == 0 {
		return errors.New("LEN=0 в полученном кадре")
	}

	body := make([]byte, lenByte)
	if _, err := io.ReadFull(rd, body); err != nil {
		return fmt.Errorf("read body (%d): %w", lenByte, err)
	}
	gotLRC, err := rd.ReadByte()
	if err != nil {
		return fmt.Errorf("read LRC: %w", err)
	}

	expectLRC := lrc(append([]byte{lenByte}, body...))
	if gotLRC != expectLRC {
		log.Printf("kkt mock: LRC mismatch got 0x%02X want 0x%02X (frame=%s)",
			gotLRC, expectLRC, hex.EncodeToString(body))
		if err := writeByte(conn, nak); err != nil {
			return fmt.Errorf("send NAK on LRC mismatch: %w", err)
		}
		return nil
	}

	twoByte := body[0] == 0xFF && len(body) >= 2
	var cmd uint16
	var data []byte
	if twoByte {
		cmd = uint16(body[0])<<8 | uint16(body[1])
		data = body[2:]
	} else {
		cmd = uint16(body[0])
		data = body[1:]
	}

	if st.cfg.dumpHex {
		log.Printf("kkt mock: rx_frame cmd=0x%04X len=%d data=%s",
			cmd, lenByte, hex.EncodeToString(data))
	} else {
		log.Printf("kkt mock: <- cmd 0x%04X (%s)", cmd, cmdName(cmd))
	}

	if err := writeByte(conn, ack); err != nil {
		return fmt.Errorf("send ACK on request: %w", err)
	}

	rspData := st.buildResponse(cmd)

	rspFrame := encodeFrame(cmd, rspData, twoByte)
	if st.cfg.dumpHex {
		log.Printf("kkt mock: tx_frame %s", hex.EncodeToString(rspFrame))
	} else {
		log.Printf("kkt mock: -> cmd 0x%04X err=0x%02X tail=%d byte(s)",
			cmd, rspData[0], len(rspData)-1)
	}

	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if _, err := conn.Write(rspFrame); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("set read deadline for ACK: %w", err)
	}
	ackByte, err := rd.ReadByte()
	if err != nil {
		return fmt.Errorf("read ACK after response: %w", err)
	}
	if ackByte != ack && st.cfg.dumpHex {
		log.Printf("kkt mock: expected ACK after response, got 0x%02X", ackByte)
	}
	return nil
}

func (st *kktState) buildResponse(cmd uint16) []byte {
	switch cmd {
	case cmdShortStatus:
		return st.respShortStatus()
	case cmdPrintString:
		return st.respPrintString()
	case cmdCloseShiftZ:
		return st.respCloseShiftZ()
	case cmdOpenShift:
		return st.respOpenShift()
	case cmdReportCalcStart:
		return st.respReportCalcStart()
	case cmdReportCalcForm:
		return st.respReportCalcForm()
	case cmdShiftParams:
		return st.respShiftParams()
	case cmdOperationV2:
		return st.respOperationV2()
	case cmdCloseReceipt:
		return st.respCloseReceipt()
	default:
		return []byte{0x42}
	}
}

func (st *kktState) respShortStatus() []byte {
	st.shift.mu.Lock()
	shiftOpen := st.shift.state == 1
	st.shift.mu.Unlock()

	var flagsLow uint16
	if shiftOpen {
		flagsLow |= 1 << 13
	}

	out := make([]byte, 0, 12)
	out = append(out, 0x00)
	out = append(out, 1)
	out = append(out, byte(flagsLow), byte(flagsLow>>8))
	out = append(out, 0)
	out = append(out, 2)
	out = append(out, 0)
	out = append(out, 0)
	out = append(out, 80)
	out = append(out, 230)
	out = append(out, 0)
	out = append(out, 0)
	return out
}

func (st *kktState) respShiftParams() []byte {
	st.shift.mu.Lock()
	state := st.shift.state
	shiftNum := st.shift.shiftNumber
	receiptNum := st.shift.receiptNumber
	st.shift.mu.Unlock()

	out := make([]byte, 0, 6)
	out = append(out, 0x00)
	out = append(out, state)
	out = append(out, byte(shiftNum), byte(shiftNum>>8))
	out = append(out, byte(receiptNum), byte(receiptNum>>8))
	return out
}

func (st *kktState) respOpenShift() []byte {
	st.shift.mu.Lock()
	st.shift.state = 1
	st.shift.shiftNumber++
	st.shift.receiptNumber = 0
	shiftNum := st.shift.shiftNumber
	st.shift.mu.Unlock()

	log.Printf("kkt mock: shift opened shift_number=%d", shiftNum)

	fd := atomic.AddUint32(&st.fdCounter, 1)
	fs := atomic.AddUint32(&st.fsCounter, 1)
	now := time.Now()

	out := make([]byte, 0, 15)
	out = append(out, 0x00)
	out = append(out, 1) // номер оператора
	out = append(out, byte(fd), byte(fd>>8), byte(fd>>16), byte(fd>>24))
	out = append(out, byte(fs), byte(fs>>8), byte(fs>>16), byte(fs>>24))
	out = append(out,
		byte(now.Year()%100),
		byte(now.Month()),
		byte(now.Day()),
		byte(now.Hour()),
		byte(now.Minute()),
	)
	return out
}

func (st *kktState) respCloseShiftZ() []byte {
	st.shift.mu.Lock()
	shiftNum := st.shift.shiftNumber
	st.shift.state = 0
	st.shift.mu.Unlock()

	log.Printf("kkt mock: shift closed (Z-report) shift_number=%d", shiftNum)

	fd := atomic.AddUint32(&st.fdCounter, 1)
	fs := atomic.AddUint32(&st.fsCounter, 1)
	now := time.Now()

	out := make([]byte, 0, 15)
	out = append(out, 0x00)
	out = append(out, 1) // номер оператора
	out = append(out, byte(fd), byte(fd>>8), byte(fd>>16), byte(fd>>24))
	out = append(out, byte(fs), byte(fs>>8), byte(fs>>16), byte(fs>>24))
	out = append(out,
		byte(now.Year()%100),
		byte(now.Month()),
		byte(now.Day()),
		byte(now.Hour()),
		byte(now.Minute()),
	)
	return out
}

func (st *kktState) respPrintString() []byte {
	// Подтверждаем без ошибок; реальная ККТ возвращает 1 байт (код ошибки=0).
	return []byte{0x00}
}

func (st *kktState) respReportCalcStart() []byte {
	return []byte{0x00}
}

func (st *kktState) respReportCalcForm() []byte {
	// Минимальный ответ: err=0, FD(4), FP(4), unconfirmed_count(3), date_exists=0.
	fd := atomic.AddUint32(&st.fdCounter, 1)
	fs := atomic.AddUint32(&st.fsCounter, 1)

	out := make([]byte, 0, 13)
	out = append(out, 0x00)
	out = append(out, byte(fd), byte(fd>>8), byte(fd>>16), byte(fd>>24))
	out = append(out, byte(fs), byte(fs>>8), byte(fs>>16), byte(fs>>24))
	out = append(out, 0, 0, 0) // unconfirmed_count = 0
	// дата первого неподтверждённого ФД отсутствует (нет байтов)
	return out
}

func (st *kktState) respOperationV2() []byte {
	if st.cfg.scenario == scenarioOperationError {
		return []byte{0x49}
	}
	return []byte{0x00}
}

func (st *kktState) respCloseReceipt() []byte {
	if st.cfg.scenario == scenarioCloseError {
		return []byte{0xA0}
	}

	fd := atomic.AddUint32(&st.fdCounter, 1)
	fs := atomic.AddUint32(&st.fsCounter, 1)

	st.shift.mu.Lock()
	st.shift.receiptNumber++
	st.shift.mu.Unlock()

	out := make([]byte, 0, 19)
	out = append(out, 0x00)
	out = append(out, 0, 0, 0, 0, 0)
	out = append(out, byte(fd), byte(fd>>8), byte(fd>>16), byte(fd>>24))
	out = append(out, byte(fs), byte(fs>>8), byte(fs>>16), byte(fs>>24))

	now := time.Now()
	out = append(out,
		byte(now.Year()%100),
		byte(now.Month()),
		byte(now.Day()),
		byte(now.Hour()),
		byte(now.Minute()),
	)
	return out
}

func encodeFrame(cmd uint16, data []byte, twoByte bool) []byte {
	var cmdBytes []byte
	if twoByte {
		cmdBytes = []byte{byte(cmd >> 8), byte(cmd)}
	} else {
		cmdBytes = []byte{byte(cmd)}
	}
	bodyLen := len(cmdBytes) + len(data)

	out := make([]byte, 0, 3+bodyLen)
	out = append(out, stx, byte(bodyLen))
	out = append(out, cmdBytes...)
	out = append(out, data...)
	out = append(out, lrc(out[1:]))
	return out
}

func lrc(b []byte) byte {
	var x byte
	for _, v := range b {
		x ^= v
	}
	return x
}

func writeByte(conn net.Conn, b byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	_, err := conn.Write([]byte{b})
	return err
}

func byteName(b byte) string {
	switch b {
	case enq:
		return "ENQ"
	case stx:
		return "STX"
	case ack:
		return "ACK"
	case nak:
		return "NAK"
	}
	return "?"
}

func cmdName(cmd uint16) string {
	switch cmd {
	case cmdShortStatus:
		return "ShortStatus"
	case cmdPrintString:
		return "PrintString"
	case cmdCloseShiftZ:
		return "CloseShiftZ"
	case cmdOpenShift:
		return "OpenShift"
	case cmdReportCalcStart:
		return "ReportCalcStart"
	case cmdReportCalcForm:
		return "ReportCalcForm"
	case cmdShiftParams:
		return "ShiftParams"
	case cmdOperationV2:
		return "OperationV2"
	case cmdCloseReceipt:
		return "CloseReceiptV2"
	}
	return "Unknown"
}

func loadEnv() {
	candidates := []string{
		"server/mocks/kkt/.env",
		"mocks/kkt/.env",
		".env",
	}
	for _, p := range candidates {
		if err := godotenv.Load(p); err == nil {
			log.Printf("kkt mock: loaded env from %s", p)
			return
		}
	}
	log.Printf("kkt mock: .env not found, using system environment")
}

func readConfig() mockConfig {
	sc := parseScenario(envStr("MOCK_KKT_SCENARIO", string(scenarioSuccess)))

	// Если MOCK_KKT_SHIFT_STATE не задан явно, сценарий задаёт начальное состояние смены.
	shiftStateEnv := strings.TrimSpace(os.Getenv("MOCK_KKT_SHIFT_STATE"))
	var initState byte
	if shiftStateEnv != "" {
		initState = parseShiftState(shiftStateEnv)
	} else {
		switch sc {
		case scenarioShiftClosed:
			initState = 0
		case scenarioShiftExpired:
			initState = 2
		default:
			initState = 1
		}
	}

	return mockConfig{
		host:           envStr("MOCK_KKT_HOST", "127.0.0.1"),
		port:           envInt("MOCK_KKT_PORT", 7778),
		scenario:       sc,
		initShiftState: initState,
		initShiftNum:   uint16(envInt("MOCK_KKT_SHIFT_NUMBER", 1)),
		initReceiptNum: uint16(envInt("MOCK_KKT_RECEIPT_NUMBER", 1)),
		dumpHex:        envBool("MOCK_KKT_DUMP_HEX", false),
	}
}

func envStr(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func envInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("kkt mock: %s=%q is not an int, using default %d", name, v, def)
		return def
	}
	return i
}

func envBool(name string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch v {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	log.Printf("kkt mock: %s=%q is not a bool, using default %v", name, v, def)
	return def
}

func parseScenario(raw string) scenario {
	s := scenario(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case scenarioSuccess, scenarioShiftClosed, scenarioShiftExpired,
		scenarioOperationError, scenarioCloseError:
		return s
	}
	log.Printf("kkt mock: unknown scenario %q, using %q", raw, scenarioSuccess)
	return scenarioSuccess
}

func parseShiftState(raw string) byte {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "closed", "0":
		return 0
	case "expired", "2":
		return 2
	case "open", "1", "":
		return 1
	}
	log.Printf("kkt mock: unknown MOCK_KKT_SHIFT_STATE=%q, using open", raw)
	return 1
}

func shiftStateName(b byte) string {
	switch b {
	case 0:
		return "closed"
	case 1:
		return "open"
	case 2:
		return "expired"
	}
	return "unknown"
}
