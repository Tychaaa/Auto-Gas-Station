package kkt

import "fmt"

// ShortStatus - распарсенный ответ команды 0x10 "Короткий запрос состояния".
// Описание полей по протоколу A.2.0.
type ShortStatus struct {
	OperatorIndex byte   // порядковый номер оператора (0..30)
	FlagsLow      uint16 // младшие 16 бит флагов ККТ
	Mode          byte   // режим ККТ (см. Приложение 1)
	Submode       byte   // подрежим ККТ
	// Дальше идут счётчики и прочее, для нашей задачи не нужны.
	BatteryVoltage byte
	PowerVoltage   byte
	Reserved       byte
	FlagsHi        byte
	Raw            []byte
}

// IsShiftOpen возвращает true, если бит "Смена открыта" (бит 13 младших флагов).
//
// По документации стандартного нижнего уровня бит 13 флагов ККТ - "Смена открыта".
func (s ShortStatus) IsShiftOpen() bool {
	const ShiftOpenBit = 1 << 13
	return s.FlagsLow&ShiftOpenBit != 0
}

// IsReceiptOpen возвращает true, если бит "Чек открыт" (бит 14).
func (s ShortStatus) IsReceiptOpen() bool {
	const ReceiptOpenBit = 1 << 14
	return s.FlagsLow&ReceiptOpenBit != 0
}

// ParseShortStatus разбирает Data из ответа команды 0x10.
// Поле Data в ответе начинается СРАЗУ ПОСЛЕ кода ошибки (код ошибки уже снят
// клиентом). Минимальный полезный размер - 11 байт.
func ParseShortStatus(data []byte) (*ShortStatus, error) {
	if len(data) < 11 {
		return nil, fmt.Errorf("ShortStatus: ожидалось >=11 байт, получили %d", len(data))
	}
	s := &ShortStatus{
		OperatorIndex:  data[0],
		FlagsLow:       uint16(data[1]) | uint16(data[2])<<8,
		Mode:           data[4],
		Submode:        data[5],
		BatteryVoltage: data[7],
		PowerVoltage:   data[8],
		FlagsHi:        data[9],
		Reserved:       data[10],
		Raw:            append([]byte(nil), data...),
	}
	return s, nil
}

// ShiftParams - распарсенный ответ команды 0xFF40 "Запрос параметров текущей смены".
//
// Формат ответа (после кода ошибки):
//
//	Состояние смены  (1 байт): 0 - закрыта, 1 - открыта, 2 - просрочена (>24ч)
//	Номер смены      (2 байта)
//	Номер чека       (2 байта)
type ShiftParams struct {
	State       byte
	ShiftNumber uint16
	ReceiptNum  uint16
}

// IsOpen возвращает true, если смена считается активной (включая просроченную - её
// тоже надо закрыть, но касса как минимум видит её как "открытую").
func (s ShiftParams) IsOpen() bool {
	return s.State == 1
}

// IsExpired возвращает true, если смена открыта более 24 часов и должна быть закрыта.
func (s ShiftParams) IsExpired() bool {
	return s.State == 2
}

// StateName - человекочитаемое имя состояния смены.
func (s ShiftParams) StateName() string {
	switch s.State {
	case 0:
		return "закрыта"
	case 1:
		return "открыта"
	case 2:
		return "просрочена (>24ч)"
	default:
		return fmt.Sprintf("неизвестно (0x%02X)", s.State)
	}
}

// ParseShiftParams разбирает Data из ответа FF40h.
func ParseShiftParams(data []byte) (*ShiftParams, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("ShiftParams: ожидалось >=5 байт, получили %d", len(data))
	}
	return &ShiftParams{
		State:       data[0],
		ShiftNumber: uint16(data[1]) | uint16(data[2])<<8,
		ReceiptNum:  uint16(data[3]) | uint16(data[4])<<8,
	}, nil
}
