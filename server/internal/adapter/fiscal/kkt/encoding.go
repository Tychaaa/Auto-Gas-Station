package kkt

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/text/encoding/charmap"
)

// PutPassword пишет пароль (4 байта, little-endian, в "двоично-десятичном" виде - но
// по факту в протоколе это просто 4 байта числа без хитростей; самый младший байт
// первый. См. примеры команд 10h, FF45h, FF46h - "Пароль (4 байта)").
func PutPassword(buf *[]byte, password uint32) {
	*buf = append(*buf,
		byte(password),
		byte(password>>8),
		byte(password>>16),
		byte(password>>24),
	)
}

// PutMoney5 пишет сумму в копейках, 5 байт, little-endian. Используется в FF45h/FF46h.
// Спец-значение 0xFFFFFFFFFF означает "не задано" (касса считает сама).
func PutMoney5(buf *[]byte, minor int64) error {
	if minor != int64(MoneyUnset) {
		if minor < 0 {
			return fmt.Errorf("сумма не может быть отрицательной: %d коп", minor)
		}
		if uint64(minor) > 0xFFFFFFFFFE {
			return fmt.Errorf("сумма %d коп выходит за пределы 5 байт", minor)
		}
	}
	u := uint64(minor)
	*buf = append(*buf,
		byte(u),
		byte(u>>8),
		byte(u>>16),
		byte(u>>24),
		byte(u>>32),
	)
	return nil
}

// MoneyUnset = 0xFFFFFFFFFF - значение "не указано", касса считает сама (FF46h).
const MoneyUnset int64 = 0xFFFFFFFFFF

// PutQuantity6 пишет количество (6 знаков после запятой, 6 байт little-endian) для FF46h.
// quantityMicro - это литры * 1_000_000 (например, 10л -> 10_000_000; 3.2л -> 3_200_000).
func PutQuantity6(buf *[]byte, quantityMicro int64) error {
	if quantityMicro < 0 {
		return fmt.Errorf("количество не может быть отрицательным: %d", quantityMicro)
	}
	if uint64(quantityMicro) > 0xFFFFFFFFFFFF {
		return fmt.Errorf("количество %d выходит за пределы 6 байт", quantityMicro)
	}
	u := uint64(quantityMicro)
	*buf = append(*buf,
		byte(u),
		byte(u>>8),
		byte(u>>16),
		byte(u>>24),
		byte(u>>32),
		byte(u>>40),
	)
	return nil
}

// PutByte пишет один байт.
func PutByte(buf *[]byte, b byte) {
	*buf = append(*buf, b)
}

// PutUint16LE пишет 2 байта little-endian.
func PutUint16LE(buf *[]byte, v uint16) {
	*buf = append(*buf, byte(v), byte(v>>8))
}

// PutString1251 кодирует строку из UTF-8 в WIN1251 и пишет ровно maxLen байт
// (если строка короче - дополняется нулями справа).
// Если pad=false и строка короче - пишется как есть.
// Если строка длиннее maxLen - возвращается ошибка.
func PutString1251(buf *[]byte, s string, maxLen int, pad bool) error {
	encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(s))
	if err != nil {
		return fmt.Errorf("не удалось закодировать строку %q в WIN1251: %w", s, err)
	}
	if len(encoded) > maxLen {
		return fmt.Errorf("строка длиной %d байт после WIN1251 не помещается в %d байт", len(encoded), maxLen)
	}
	*buf = append(*buf, encoded...)
	if pad {
		for i := len(encoded); i < maxLen; i++ {
			*buf = append(*buf, 0x00)
		}
	}
	return nil
}

// ReadMoney5 читает 5 байт little-endian как сумму в копейках.
func ReadMoney5(p []byte) (int64, error) {
	if len(p) < 5 {
		return 0, errors.New("ReadMoney5: нужно 5 байт")
	}
	u := uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 | uint64(p[4])<<32
	return int64(u), nil
}

// ReadUint16LE читает 2 байта little-endian как uint16.
func ReadUint16LE(p []byte) (uint16, error) {
	if len(p) < 2 {
		return 0, errors.New("ReadUint16LE: нужно 2 байта")
	}
	return uint16(p[0]) | uint16(p[1])<<8, nil
}

// ReadUint32LE читает 4 байта little-endian как uint32.
func ReadUint32LE(p []byte) (uint32, error) {
	if len(p) < 4 {
		return 0, errors.New("ReadUint32LE: нужно 4 байта")
	}
	return uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24, nil
}

// ReadDateTime5 читает 5 байт DATE_TIME (YY MM DD hh mm) и собирает time.Time.
// Год передаётся как 2 цифры от 2000 года.
func ReadDateTime5(p []byte) (time.Time, error) {
	if len(p) < 5 {
		return time.Time{}, errors.New("ReadDateTime5: нужно 5 байт")
	}
	year := 2000 + int(p[0])
	month := int(p[1])
	day := int(p[2])
	hour := int(p[3])
	minute := int(p[4])
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("некорректная дата %02d-%02d-%02d %02d:%02d", year, month, day, hour, minute)
	}
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local), nil
}
