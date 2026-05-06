// Пакет kkt реализует клиент к ККТ PayOnline-01-ФА по протоколу A.2.0
// (стандартный нижний уровень: STX/LRC + ENQ/ACK/NAK) поверх TCP.
package kkt

// Служебные байты стандартного нижнего уровня (см. документ "Протокол ККТ v.A.2.0",
// раздел "Стандартный нижний уровень").
const (
	ENQ byte = 0x05
	STX byte = 0x02
	ACK byte = 0x06
	NAK byte = 0x15
)

// CommandCode - код команды. Однобайтовая (как 0x10) или двухбайтовая (как 0xFF45)
// команда хранится в одном uint16 в порядке "первым идёт старший байт", т.е.
// для команды 0xFF45 значение = 0xFF45, для 0x10 значение = 0x0010.
type CommandCode uint16

// Bytes возвращает кодирование команды для сериализации в кадр (старший идёт первым).
func (c CommandCode) Bytes() []byte {
	if c <= 0xFF {
		return []byte{byte(c)}
	}
	return []byte{byte(c >> 8), byte(c)}
}

// Hex возвращает удобное представление "0xFF45" / "0x10".
func (c CommandCode) Hex() string {
	if c <= 0xFF {
		return formatHex16("0x", uint16(c), 2)
	}
	return formatHex16("0x", uint16(c), 4)
}

// Список поддерживаемых команд в этом клиенте.
const (
	CmdShortStatus  CommandCode = 0x10   // Короткий запрос состояния
	CmdLongStatus   CommandCode = 0x11   // Запрос состояния ККТ (длинный)
	CmdSendTLV      CommandCode = 0xFF0C // Передать произвольную TLV структуру
	CmdShiftParams  CommandCode = 0xFF40 // Запрос параметров текущей смены
	CmdCloseReceipt CommandCode = 0xFF45 // Закрытие чека расширенное вариант №2
	CmdOperationV2  CommandCode = 0xFF46 // Операция V2
)

// Тип операции для FF46h.
const (
	OpSale       byte = 1 // Приход
	OpSaleReturn byte = 2 // Возврат прихода
	OpExpense    byte = 3 // Расход
	OpExpenseRet byte = 4 // Возврат расхода
)

// Известные TLV-теги ФФД, которые мы используем.
const (
	TagAdditionalReceiptAttribute uint16 = 1192 // дополнительный реквизит чека
)

// Имя команды для логов.
func (c CommandCode) Name() string {
	switch c {
	case CmdShortStatus:
		return "Короткий запрос состояния"
	case CmdLongStatus:
		return "Запрос состояния ККТ"
	case CmdSendTLV:
		return "Передать TLV"
	case CmdShiftParams:
		return "Запрос параметров текущей смены"
	case CmdCloseReceipt:
		return "Закрытие чека V2"
	case CmdOperationV2:
		return "Операция V2"
	default:
		return "Команда " + c.Hex()
	}
}

// formatHex16 - вспомогательная hex-форматировалка без зависимости от fmt в этом файле.
func formatHex16(prefix string, v uint16, width int) string {
	const hex = "0123456789ABCDEF"
	buf := make([]byte, 0, len(prefix)+width)
	buf = append(buf, prefix...)
	for i := width - 1; i >= 0; i-- {
		buf = append(buf, hex[(v>>(uint(i)*4))&0x0F])
	}
	return string(buf)
}
