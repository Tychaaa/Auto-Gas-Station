package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	aztDEL = 0x7F
	aztSTX = 0x02
	aztETX = 0x03
	aztACK = 0x06
	aztNAK = 0x15
	aztCAN = 0x18
)

type serialConfig struct {
	Port     string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
	Address  int
}

type emulatorConfig struct {
	serial      serialConfig
	Scenario    string
	StepLiters  float64
	DefaultPrice int64
}

type emulatorState struct {
	statusCode   byte
	reasonCode   byte
	priceMinor   int64
	doseLiters   int64
	currentLiters int64
	transaction  int64
	scenario     string
	stepHundredths int64
}

func main() {
	loadEnv()

	cfg := emulatorConfig{
		serial: serialConfig{
			Port:     envString("MOCK_FUEL_PORT", "COM2"),
			Baud:     envInt("MOCK_FUEL_BAUD", 4800),
			DataBits: envInt("MOCK_FUEL_DATABITS", 7),
			StopBits: envInt("MOCK_FUEL_STOPBITS", 2),
			Parity:   envString("MOCK_FUEL_PARITY", "even"),
			Address:  envInt("MOCK_FUEL_ADDRESS", 1),
		},
		Scenario:     strings.ToLower(strings.TrimSpace(envString("MOCK_FUEL_SCENARIO", "normal"))),
		StepLiters:   envFloat("MOCK_FUEL_STEP_LITERS", 0.25),
		DefaultPrice: int64(envInt("MOCK_FUEL_DEFAULT_PRICE_MINOR", 5500)),
	}

	port, err := openSerial(cfg.serial)
	if err != nil {
		log.Fatalf("fuel emulator open serial failed: %v", err)
	}
	defer port.Close()

	state := &emulatorState{
		statusCode:      '0',
		reasonCode:      '0',
		priceMinor:      cfg.DefaultPrice,
		scenario:        cfg.Scenario,
		stepHundredths:  int64(math.Round(cfg.StepLiters * 100)),
	}
	if state.stepHundredths <= 0 {
		state.stepHundredths = 25
	}

	log.Printf("fuel emulator started on %s address=%d scenario=%s", cfg.serial.Port, cfg.serial.Address, cfg.Scenario)

	buffer := make([]byte, 256)
	packet := make([]byte, 0, 64)
	for {
		n, err := port.Read(buffer)
		if err != nil {
			log.Fatalf("fuel emulator read failed: %v", err)
		}
		if n == 0 {
			continue
		}

		packet = append(packet, buffer[:n]...)
		if !isPacketComplete(packet) {
			continue
		}

		response, err := handlePacket(packet, state, cfg.serial.Address)
		packet = packet[:0]
		if err != nil {
			log.Printf("fuel emulator packet error: %v", err)
			continue
		}
		if len(response) == 0 {
			continue
		}
		if _, err := port.Write(response); err != nil {
			log.Fatalf("fuel emulator write failed: %v", err)
		}
	}
}

func handlePacket(raw []byte, state *emulatorState, address int) ([]byte, error) {
	payload, err := decodePayload(raw)
	if err != nil {
		return nil, err
	}
	if len(payload) < 2 {
		return nil, errors.New("request payload is too short")
	}

	expectedAddress := byte(0x20 + address)
	if payload[0] != expectedAddress {
		return nil, nil
	}

	command := payload[1]
	data := payload[2:]
	advanceProgress(state)

	switch command {
	case '1':
		response := []byte{state.statusCode}
		if state.statusCode == '4' {
			response = append(response, state.reasonCode)
		}
		return encodeDataResponse(response), nil
	case 'Q':
		if state.statusCode != '0' && state.statusCode != '1' {
			return encodeShortResponse(aztCAN), nil
		}
		priceMinor, err := decodeDigits(data)
		if err != nil {
			return encodeShortResponse(aztNAK), nil
		}
		state.priceMinor = priceMinor
		return encodeShortResponse(aztACK), nil
	case 'S':
		if state.statusCode != '0' && state.statusCode != '1' {
			return encodeShortResponse(aztCAN), nil
		}
		amountMinor, err := decodeDigits(data)
		if err != nil || state.priceMinor <= 0 {
			return encodeShortResponse(aztCAN), nil
		}
		state.doseLiters = amountMinor * 100 / state.priceMinor
		state.currentLiters = 0
		state.reasonCode = '0'
		return encodeShortResponse(aztACK), nil
	case 'T':
		if state.statusCode != '0' && state.statusCode != '1' {
			return encodeShortResponse(aztCAN), nil
		}
		liters, err := decodeDigits(data)
		if err != nil {
			return encodeShortResponse(aztNAK), nil
		}
		state.doseLiters = liters
		state.currentLiters = 0
		state.reasonCode = '0'
		return encodeShortResponse(aztACK), nil
	case '2':
		if state.statusCode != '0' && state.statusCode != '1' {
			return encodeShortResponse(aztCAN), nil
		}
		state.statusCode = '2'
		state.reasonCode = '0'
		state.currentLiters = 0
		state.transaction++
		if state.scenario == "timeout" {
			return nil, nil
		}
		return encodeShortResponse(aztACK), nil
	case '4':
		response, err := encodeDigits(state.currentLiters, 5)
		if err != nil {
			return encodeShortResponse(aztCAN), nil
		}
		return encodeDataResponse(append([]byte{'0'}, response...)), nil
	case '5':
		if state.statusCode != '4' {
			return encodeShortResponse(aztCAN), nil
		}
		amountMinor := state.currentLiters * state.priceMinor / 100
		litersDigits, _ := encodeDigits(state.currentLiters, 6)
		amountDigits, _ := encodeDigits(amountMinor, 8)
		priceDigits, _ := encodeDigits(state.priceMinor, 4)
		payload := append(litersDigits, amountDigits...)
		payload = append(payload, priceDigits...)
		return encodeDataResponse(payload), nil
	case '8':
		if state.statusCode != '4' {
			return encodeShortResponse(aztCAN), nil
		}
		state.statusCode = '0'
		state.reasonCode = '0'
		state.currentLiters = 0
		state.doseLiters = 0
		return encodeShortResponse(aztACK), nil
	case 'P':
		return encodeDataResponse([]byte("00000002")), nil
	case 'X':
		doseDigits, _ := encodeDigits(state.doseLiters, 5)
		return encodeDataResponse(append([]byte{'0'}, doseDigits...)), nil
	case 'Y':
		trDigits, _ := encodeDigits(state.transaction, 8)
		return encodeDataResponse(trDigits), nil
	default:
		return encodeShortResponse(aztNAK), nil
	}
}

func advanceProgress(state *emulatorState) {
	switch state.statusCode {
	case '2':
		state.statusCode = '3'
	case '3':
		target := state.doseLiters
		if state.scenario == "partial" && target > state.stepHundredths {
			target -= state.stepHundredths
		}
		state.currentLiters += state.stepHundredths
		if state.currentLiters >= target {
			state.currentLiters = target
			state.statusCode = '4'
			if state.scenario == "partial" {
				state.reasonCode = '1'
			} else {
				state.reasonCode = '0'
			}
		}
	}
}

func loadEnv() {
	candidates := []string{
		"server/mocks/fuel-dispenser/.env",
		"mocks/fuel-dispenser/.env",
		".env",
	}
	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			log.Printf("fuel emulator: loaded env from %s", path)
			return
		}
	}
	log.Printf("fuel emulator: .env not found, using system environment")
}

func openSerial(cfg serialConfig) (*os.File, error) {
	path := cfg.Port
	if !strings.HasPrefix(path, `\\.\`) {
		path = `\\.\` + strings.TrimSpace(path)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open serial %s: %w", cfg.Port, err)
	}
	return file, nil
}

func decodePayload(raw []byte) ([]byte, error) {
	if len(raw) < 5 {
		return nil, errors.New("packet is too short")
	}
	startIdx := bytes.IndexByte(raw, aztSTX)
	if startIdx < 0 {
		return nil, errors.New("packet does not contain STX")
	}
	packet := raw[startIdx:]
	checksum := packet[len(packet)-1]
	body := packet[1 : len(packet)-3]
	if len(body)%2 != 0 {
		return nil, errors.New("packet body has odd length")
	}

	data := make([]byte, 0, len(body)/2)
	for i := 0; i < len(body); i += 2 {
		value := body[i]
		if body[i+1] != ^value {
			return nil, fmt.Errorf("complement mismatch at %d", i/2)
		}
		data = append(data, value)
	}

	if checksum != calculateChecksum(data) {
		return nil, errors.New("checksum mismatch")
	}
	return data, nil
}

func encodeDataResponse(data []byte) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 2+len(data)*2+3))
	buf.WriteByte(aztDEL)
	buf.WriteByte(aztSTX)
	for _, b := range data {
		buf.WriteByte(b)
		buf.WriteByte(^b)
	}
	buf.WriteByte(aztETX)
	buf.WriteByte(aztETX)
	buf.WriteByte(calculateChecksum(data))
	return buf.Bytes()
}

func encodeShortResponse(code byte) []byte {
	return []byte{aztDEL, code}
}

func calculateChecksum(data []byte) byte {
	checksum := byte(0)
	for _, b := range data {
		checksum ^= b
	}
	checksum ^= aztETX
	checksum |= 0x40
	return checksum
}

func isPacketComplete(raw []byte) bool {
	if len(raw) < 2 {
		return false
	}
	if raw[0] == aztDEL && len(raw) == 2 {
		switch raw[1] {
		case aztACK, aztNAK, aztCAN:
			return true
		}
	}
	startIdx := bytes.IndexByte(raw, aztSTX)
	if startIdx < 0 {
		return false
	}
	packet := raw[startIdx:]
	return len(packet) >= 5 && packet[len(packet)-3] == aztETX && packet[len(packet)-2] == aztETX
}

func decodeDigits(data []byte) (int64, error) {
	var result int64
	for _, b := range data {
		if b < '0' || b > '9' {
			return 0, fmt.Errorf("invalid digit %q", b)
		}
		result = result*10 + int64(b-'0')
	}
	return result, nil
}

func encodeDigits(value int64, width int) ([]byte, error) {
	if value < 0 {
		return nil, errors.New("value must be non-negative")
	}
	format := fmt.Sprintf("%%0%dd", width)
	text := fmt.Sprintf(format, value)
	if len(text) > width {
		return nil, fmt.Errorf("value %d does not fit width %d", value, width)
	}
	return []byte(text), nil
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(name string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
