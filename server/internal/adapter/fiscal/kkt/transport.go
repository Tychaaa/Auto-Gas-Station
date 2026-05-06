package kkt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"
)

// Transport - сетевой клиент стандартного нижнего уровня поверх TCP.
// Не потокобезопасен, рассчитан на использование одной горутиной на одну сессию.
type Transport struct {
	conn         net.Conn
	log          *slog.Logger
	dumpHex      bool
	readTimeout  time.Duration
	byteTimeout  time.Duration
	ackTimeout   time.Duration
	connTimeout  time.Duration
	maxResendAck int
}

// TransportOptions - параметры подключения.
type TransportOptions struct {
	Address        string
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	ByteTimeout    time.Duration
	AckTimeout     time.Duration
	DumpHex        bool
	Logger         *slog.Logger
}

// Dial устанавливает TCP-соединение и инициирует синхронизацию по нижнему уровню (ENQ).
func Dial(ctx context.Context, opts TransportOptions) (*Transport, error) {
	if opts.Logger == nil {
		return nil, errors.New("Logger обязателен")
	}
	d := net.Dialer{Timeout: opts.ConnectTimeout}
	opts.Logger.Info("kkt.dial", slog.String("addr", opts.Address))
	conn, err := d.DialContext(ctx, "tcp", opts.Address)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к ККТ %s: %w", opts.Address, err)
	}

	t := &Transport{
		conn:         conn,
		log:          opts.Logger,
		dumpHex:      opts.DumpHex,
		readTimeout:  opts.ReadTimeout,
		byteTimeout:  opts.ByteTimeout,
		ackTimeout:   opts.AckTimeout,
		connTimeout:  opts.ConnectTimeout,
		maxResendAck: 3,
	}
	if err := t.handshake(); err != nil {
		conn.Close()
		return nil, err
	}
	return t, nil
}

// Close закрывает соединение.
func (t *Transport) Close() error {
	if t == nil || t.conn == nil {
		return nil
	}
	return t.conn.Close()
}

// handshake посылает ENQ и ждёт NAK (ККТ свободна) или ACK (ККТ занята / есть
// несчитанный ответ - сбрасываем его и пробуем ещё раз).
func (t *Transport) handshake() error {
	for attempt := 1; attempt <= 5; attempt++ {
		if err := t.writeByte(ENQ, "ENQ"); err != nil {
			return fmt.Errorf("handshake: %w", err)
		}
		b, err := t.readByte(t.byteTimeout)
		if err != nil {
			return fmt.Errorf("handshake: нет ответа на ENQ: %w", err)
		}
		t.logByte("rx_byte", b)
		switch b {
		case NAK:
			t.log.Info("kkt.handshake.idle")
			return nil
		case ACK:
			t.log.Warn("kkt.handshake.pending_response_drained")
			if _, err := t.readResponseFrame(false); err != nil {
				t.log.Warn("kkt.handshake.drain_failed", slog.String("err", err.Error()))
			} else {
				_ = t.writeByte(ACK, "ACK")
			}
			continue
		default:
			t.log.Warn("kkt.handshake.unexpected_byte", slog.String("hex", HexDump([]byte{b})))
		}
	}
	return errors.New("handshake: касса не отвечает корректно после 5 попыток")
}

// Exchange отправляет команду и ждёт ответ. cmd - код команды для логов и для
// определения, ожидать ли 2-байтовый код в ответе.
func (t *Transport) Exchange(cmd CommandCode, payload []byte) (*Frame, error) {
	frame, err := EncodeFrame(cmd, payload)
	if err != nil {
		return nil, err
	}

	t.log.Info("kkt.tx",
		slog.String("cmd", cmd.Hex()),
		slog.String("name", cmd.Name()),
		slog.Int("body_len", int(frame[1])),
		slog.Int("frame_len", len(frame)),
	)
	t.logBytes("tx_bytes", frame)

	start := time.Now()

	for attempt := 1; attempt <= t.maxResendAck; attempt++ {
		if err := t.writeAll(frame); err != nil {
			return nil, fmt.Errorf("send frame: %w", err)
		}
		ack, err := t.readByte(t.ackTimeout)
		if err != nil {
			return nil, fmt.Errorf("ожидание ACK после команды %s: %w", cmd.Hex(), err)
		}
		t.logByte("rx_byte_ack", ack)
		switch ack {
		case ACK:
			twoByte := cmd > 0xFF
			rsp, err := t.readResponseFrame(twoByte)
			if err != nil {
				return nil, fmt.Errorf("чтение ответа на %s: %w", cmd.Hex(), err)
			}
			if err := t.writeByte(ACK, "ACK"); err != nil {
				return nil, fmt.Errorf("отправка ACK после ответа: %w", err)
			}
			t.log.Info("kkt.rx",
				slog.String("cmd", rsp.Cmd.Hex()),
				slog.String("name", rsp.Cmd.Name()),
				slog.Int("data_len", len(rsp.Data)),
				slog.Duration("elapsed", time.Since(start)),
			)
			return rsp, nil
		case NAK:
			t.log.Warn("kkt.tx.nak_retry",
				slog.Int("attempt", attempt),
				slog.String("cmd", cmd.Hex()),
			)
			if err := t.writeByte(ENQ, "ENQ"); err != nil {
				return nil, err
			}
			b, err := t.readByte(t.byteTimeout)
			if err != nil {
				return nil, fmt.Errorf("после NAK ждём NAK на ENQ: %w", err)
			}
			if b != NAK {
				return nil, fmt.Errorf("после NAK ожидался NAK на ENQ, получили 0x%02X", b)
			}
			continue
		default:
			return nil, fmt.Errorf("неожиданный байт после команды %s: 0x%02X", cmd.Hex(), ack)
		}
	}
	return nil, fmt.Errorf("не удалось доставить команду %s после %d попыток", cmd.Hex(), t.maxResendAck)
}

// readResponseFrame читает STX | LEN | BODY(LEN) | LRC.
func (t *Transport) readResponseFrame(twoByteCmd bool) (*Frame, error) {
	for i := 0; i < 16; i++ {
		b, err := t.readByte(t.readTimeout)
		if err != nil {
			return nil, fmt.Errorf("ожидание STX: %w", err)
		}
		if b == STX {
			t.logByte("rx_stx", b)
			break
		}
		t.log.Warn("kkt.rx.skip_pre_stx", slog.String("hex", HexDump([]byte{b})))
		if i == 15 {
			return nil, errors.New("STX не найден среди первых 16 байт")
		}
	}
	lenByte, err := t.readByte(t.byteTimeout)
	if err != nil {
		return nil, fmt.Errorf("ожидание LEN: %w", err)
	}
	bodyLen := int(lenByte)
	if bodyLen == 0 {
		return nil, errors.New("LEN=0 в ответе")
	}
	rest := make([]byte, bodyLen+1)
	if err := t.readFull(rest, t.readTimeout); err != nil {
		return nil, fmt.Errorf("чтение тела ответа (%d байт + LRC): %w", bodyLen, err)
	}
	body := rest[:bodyLen]
	gotLrc := rest[bodyLen]
	expectLrc := lrc(append([]byte{lenByte}, body...))

	full := append([]byte{STX, lenByte}, rest...)
	t.logBytes("rx_bytes", full)

	if gotLrc != expectLrc {
		_ = t.writeByte(NAK, "NAK")
		return nil, fmt.Errorf("LRC не совпал: got 0x%02X want 0x%02X", gotLrc, expectLrc)
	}
	return DecodeBody(body, twoByteCmd)
}

func (t *Transport) readByte(timeout time.Duration) (byte, error) {
	if err := t.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return 0, err
	}
	buf := make([]byte, 1)
	n, err := io.ReadFull(t.conn, buf)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, io.ErrUnexpectedEOF
	}
	return buf[0], nil
}

func (t *Transport) readFull(buf []byte, timeout time.Duration) error {
	if err := t.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	_, err := io.ReadFull(t.conn, buf)
	return err
}

func (t *Transport) writeByte(b byte, name string) error {
	if err := t.conn.SetWriteDeadline(time.Now().Add(t.byteTimeout)); err != nil {
		return err
	}
	_, err := t.conn.Write([]byte{b})
	if err != nil {
		return err
	}
	if t.dumpHex {
		t.log.Debug("kkt.tx_byte",
			slog.String("name", name),
			slog.String("hex", HexDump([]byte{b})),
		)
	}
	return nil
}

func (t *Transport) writeAll(buf []byte) error {
	if err := t.conn.SetWriteDeadline(time.Now().Add(t.ackTimeout)); err != nil {
		return err
	}
	_, err := t.conn.Write(buf)
	return err
}

func (t *Transport) logByte(name string, b byte) {
	if !t.dumpHex {
		return
	}
	desc := byteName(b)
	t.log.Debug("kkt."+name,
		slog.String("hex", HexDump([]byte{b})),
		slog.String("meaning", desc),
	)
}

func (t *Transport) logBytes(name string, buf []byte) {
	if !t.dumpHex {
		return
	}
	t.log.Debug("kkt."+name,
		slog.Int("len", len(buf)),
		slog.String("hex", HexDump(buf)),
	)
}

func byteName(b byte) string {
	switch b {
	case ENQ:
		return "ENQ"
	case STX:
		return "STX"
	case ACK:
		return "ACK"
	case NAK:
		return "NAK"
	default:
		return ""
	}
}

// HexDump форматирует байты как "AA BB CC ..." для удобного чтения в логе.
func HexDump(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(b) * 3)
	const hexChars = "0123456789ABCDEF"
	for i, x := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteByte(hexChars[x>>4])
		sb.WriteByte(hexChars[x&0x0F])
	}
	return sb.String()
}
