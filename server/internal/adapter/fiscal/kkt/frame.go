package kkt

import (
	"errors"
	"fmt"
)

// Frame - распарсенный кадр стандартного нижнего уровня.
//
//	[STX] [LEN] [CMD...] [DATA...] [LRC]
//
// Где LEN = len(CMD) + len(DATA), LRC = XOR всех байт кроме STX.
// Для нашего использования CMD - это либо 1 байт (если cmd <= 0xFF), либо 2 байта
// (для команд вида 0xFFXX). Для двухбайтовой команды первый байт 0xFF.
type Frame struct {
	Cmd  CommandCode
	Data []byte
}

// EncodeFrame собирает кадр для отправки в ККТ. Возвращает полный буфер
// от STX до LRC включительно.
func EncodeFrame(cmd CommandCode, data []byte) ([]byte, error) {
	cmdBytes := cmd.Bytes()
	bodyLen := len(cmdBytes) + len(data)
	if bodyLen <= 0 || bodyLen > 0xFF {
		return nil, fmt.Errorf("длина тела сообщения %d вне диапазона 1..255", bodyLen)
	}
	out := make([]byte, 0, 3+bodyLen)
	out = append(out, STX, byte(bodyLen))
	out = append(out, cmdBytes...)
	out = append(out, data...)
	out = append(out, lrc(out[1:])) // LRC по всем байтам кроме STX (т.е. начиная с LEN)
	return out, nil
}

// DecodeBody принимает уже отделённые от транспорта байты "[LEN] [CMD] [DATA]"
// (т.е. без STX и без LRC) и опциональный признак двухбайтовой команды.
// Возвращает Frame.
func DecodeBody(body []byte, twoByteCmd bool) (*Frame, error) {
	if len(body) < 1 {
		return nil, errors.New("пустое тело кадра")
	}
	if twoByteCmd {
		if len(body) < 2 {
			return nil, errors.New("ожидался 2-байтовый код команды")
		}
		cmd := CommandCode(uint16(body[0])<<8 | uint16(body[1]))
		return &Frame{Cmd: cmd, Data: append([]byte(nil), body[2:]...)}, nil
	}
	cmd := CommandCode(body[0])
	return &Frame{Cmd: cmd, Data: append([]byte(nil), body[1:]...)}, nil
}

// IsTwoByteCommand определяет по первому байту, ожидается ли 2-байтовый код команды.
// В протоколе двухбайтовые команды начинаются с 0xFF.
func IsTwoByteCommand(firstByte byte) bool {
	return firstByte == 0xFF
}

// lrc вычисляет LRC = XOR всех байт.
func lrc(b []byte) byte {
	var x byte
	for _, v := range b {
		x ^= v
	}
	return x
}
