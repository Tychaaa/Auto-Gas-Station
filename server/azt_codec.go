package main

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	aztDEL = 0x7F
	aztSTX = 0x02
	aztETX = 0x03
	aztACK = 0x06
	aztNAK = 0x15
	aztCAN = 0x18
)

const (
	aztCmdStatus            = byte('1')
	aztCmdAuthorize         = byte('2')
	aztCmdReset             = byte('3')
	aztCmdCurrentVolume     = byte('4')
	aztCmdTotals            = byte('5')
	aztCmdConfirmTotals     = byte('8')
	aztCmdProtocolVersion   = byte('P')
	aztCmdSetPrice          = byte('Q')
	aztCmdSetAmountDose     = byte('S')
	aztCmdSetLitersDose     = byte('T')
	aztCmdTransactionNumber = byte('Y')
	aztCmdReadDose          = byte('X')
)

type AZTShortResponse byte

const (
	AZTShortResponseACK AZTShortResponse = aztACK
	AZTShortResponseNAK AZTShortResponse = aztNAK
	AZTShortResponseCAN AZTShortResponse = aztCAN
)

type AZTRequest struct {
	StartByte byte
	Address   byte
	Command   byte
	Data      []byte
}

type AZTResponse struct {
	ShortResponse *AZTShortResponse
	Data          []byte
}

func DecodeAZTPayload(raw []byte) ([]byte, error) {
	if len(raw) < 5 {
		return nil, errors.New("azt packet is too short")
	}

	startIdx := bytes.IndexByte(raw, aztSTX)
	if startIdx < 0 {
		return nil, errors.New("azt packet does not contain STX")
	}

	packet := raw[startIdx:]
	if len(packet) < 5 || packet[len(packet)-3] != aztETX || packet[len(packet)-2] != aztETX {
		return nil, errors.New("azt packet missing ETX trailer")
	}

	checksum := packet[len(packet)-1]
	body := packet[1 : len(packet)-3]
	if len(body)%2 != 0 {
		return nil, errors.New("azt packet body has odd length")
	}

	data := make([]byte, 0, len(body)/2)
	for i := 0; i < len(body); i += 2 {
		value := body[i]
		if body[i+1] != aztComplement(value) {
			return nil, fmt.Errorf("azt complement mismatch at position %d", i/2)
		}
		data = append(data, value)
	}

	if checksum != calculateAZTChecksum(data) {
		return nil, errors.New("azt checksum mismatch")
	}

	return data, nil
}

func EncodeAZTRequest(req AZTRequest) ([]byte, error) {
	if req.StartByte == 0 {
		req.StartByte = aztSTX
	}
	if req.Command == 0 {
		return nil, errors.New("azt command is required")
	}

	var payload []byte
	if req.Address != 0 {
		payload = append(payload, req.Address)
	}
	payload = append(payload, req.Command)
	payload = append(payload, req.Data...)

	buf := bytes.NewBuffer(make([]byte, 0, 1+len(payload)*2+3))
	buf.WriteByte(aztDEL)
	buf.WriteByte(req.StartByte)
	for _, b := range payload {
		buf.WriteByte(b)
		buf.WriteByte(aztComplement(b))
	}
	buf.WriteByte(aztETX)
	buf.WriteByte(aztETX)
	buf.WriteByte(calculateAZTChecksum(payload))
	return buf.Bytes(), nil
}

func EncodeAZTDataResponse(data []byte) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 2+len(data)*2+3))
	buf.WriteByte(aztDEL)
	buf.WriteByte(aztSTX)
	for _, b := range data {
		buf.WriteByte(b)
		buf.WriteByte(aztComplement(b))
	}
	buf.WriteByte(aztETX)
	buf.WriteByte(aztETX)
	buf.WriteByte(calculateAZTChecksum(data))
	return buf.Bytes()
}

func EncodeAZTShortResponse(code AZTShortResponse) []byte {
	return []byte{aztDEL, byte(code)}
}

func DecodeAZTResponse(raw []byte) (AZTResponse, error) {
	if len(raw) < 2 {
		return AZTResponse{}, errors.New("azt response is too short")
	}
	if raw[0] == aztDEL && len(raw) == 2 {
		switch raw[1] {
		case aztACK, aztNAK, aztCAN:
			code := AZTShortResponse(raw[1])
			return AZTResponse{ShortResponse: &code}, nil
		}
	}

	data, err := DecodeAZTPayload(raw)
	if err != nil {
		return AZTResponse{}, err
	}
	return AZTResponse{Data: data}, nil
}

func calculateAZTChecksum(data []byte) byte {
	checksum := byte(0)
	for _, b := range data {
		checksum ^= b
	}
	checksum ^= aztETX
	checksum |= 0x40
	return checksum
}

// aztComplement возвращает 7-битный комплиментарный байт по протоколу АЗТ 2.0:
// инвертируются только биты 0..6, бит 7 всегда равен 0. См. разд. 4 спецификации.
func aztComplement(b byte) byte {
	return b ^ 0x7F
}

func encodeDigits(value int64, width int) ([]byte, error) {
	if width <= 0 {
		return nil, errors.New("width must be positive")
	}
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

func isAZTPacketComplete(raw []byte) bool {
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
