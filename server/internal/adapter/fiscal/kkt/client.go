package kkt

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Client - высокоуровневый клиент к ККТ. Обёртка над Transport.
type Client struct {
	tr               *Transport
	sysadminPassword uint32
	operatorPassword uint32
	log              *slog.Logger
}

// ClientOptions - параметры для NewClient.
type ClientOptions struct {
	Transport        *Transport
	SysadminPassword uint32
	OperatorPassword uint32
	Logger           *slog.Logger
}

// NewClient создаёт клиента поверх уже установленного Transport.
func NewClient(opts ClientOptions) *Client {
	return &Client{
		tr:               opts.Transport,
		sysadminPassword: opts.SysadminPassword,
		operatorPassword: opts.OperatorPassword,
		log:              opts.Logger,
	}
}

// extractError извлекает первый байт (код ошибки) из ответа и возвращает его.
// Если код != 0, возвращает *KKTError. Возвращает также "хвост" Data без байта ошибки.
func extractError(cmd CommandCode, frame *Frame) ([]byte, error) {
	if len(frame.Data) < 1 {
		return nil, fmt.Errorf("ответ %s пустой", cmd.Hex())
	}
	code := frame.Data[0]
	if code != 0 {
		return frame.Data[1:], &KKTError{Cmd: cmd, Code: code}
	}
	return frame.Data[1:], nil
}

// ShortStatus - команда 0x10. Возвращает разобранный статус ККТ.
func (c *Client) ShortStatus(ctx context.Context) (*ShortStatus, error) {
	_ = ctx
	var payload []byte
	PutPassword(&payload, c.operatorPassword)
	rsp, err := c.tr.Exchange(CmdShortStatus, payload)
	if err != nil {
		return nil, err
	}
	tail, err := extractError(CmdShortStatus, rsp)
	if err != nil {
		return nil, err
	}
	return ParseShortStatus(tail)
}

// ShiftParams - команда 0xFF40. Возвращает состояние и номер смены.
func (c *Client) ShiftParams(ctx context.Context) (*ShiftParams, error) {
	_ = ctx
	var payload []byte
	PutPassword(&payload, c.sysadminPassword)
	rsp, err := c.tr.Exchange(CmdShiftParams, payload)
	if err != nil {
		return nil, err
	}
	tail, err := extractError(CmdShiftParams, rsp)
	if err != nil {
		return nil, err
	}
	return ParseShiftParams(tail)
}

// SendTLV - команда 0xFF0C. Отправляет одну TLV-структуру (тег+значение).
func (c *Client) SendTLV(ctx context.Context, tag uint16, value []byte) error {
	_ = ctx
	if len(value) > 0xFFFF {
		return fmt.Errorf("значение TLV слишком длинное: %d байт", len(value))
	}
	if 4+len(value) > 250 {
		return fmt.Errorf("TLV структура (%d байт) превышает лимит 250 байт", 4+len(value))
	}

	var payload []byte
	PutPassword(&payload, c.sysadminPassword)
	PutUint16LE(&payload, tag)
	PutUint16LE(&payload, uint16(len(value)))
	payload = append(payload, value...)

	rsp, err := c.tr.Exchange(CmdSendTLV, payload)
	if err != nil {
		return err
	}
	if _, err := extractError(CmdSendTLV, rsp); err != nil {
		return err
	}
	return nil
}

// SendTLVString - вспомогательный метод, кодирует строку в WIN1251.
func (c *Client) SendTLVString(ctx context.Context, tag uint16, value string) error {
	var buf []byte
	if err := PutString1251(&buf, value, 0xFFFF, false); err != nil {
		return err
	}
	return c.SendTLV(ctx, tag, buf)
}

// OperationV2Input - параметры команды 0xFF46.
type OperationV2Input struct {
	OperationType  byte // OpSale и т.п.
	QuantityMicro  int64
	UnitPriceMinor int64
	TotalMinor     int64 // если 0 - кодируется как MoneyUnset (касса посчитает сама)
	TaxAmountMinor int64 // если 0 - кодируется как MoneyUnset
	VATCode        byte  // 0x01..0x88
	Department     byte  // 0..16
	PaymentMethod  byte  // признак способа расчёта (тег 1214)
	PaymentSubject byte  // признак предмета расчёта (тег 1212)
	GoodName       string
}

// OperationV2 - команда 0xFF46. Возвращает только nil/error (касса в ответе шлёт лишь код ошибки).
func (c *Client) OperationV2(ctx context.Context, in OperationV2Input) error {
	_ = ctx
	if in.OperationType == 0 {
		return fmt.Errorf("OperationV2: OperationType=0")
	}
	if in.VATCode == 0 {
		return fmt.Errorf("OperationV2: VATCode=0")
	}

	var payload []byte
	PutPassword(&payload, c.operatorPassword)
	PutByte(&payload, in.OperationType)
	if err := PutQuantity6(&payload, in.QuantityMicro); err != nil {
		return err
	}
	if err := PutMoney5(&payload, in.UnitPriceMinor); err != nil {
		return err
	}
	total := in.TotalMinor
	if total == 0 {
		total = MoneyUnset
	}
	if err := PutMoney5(&payload, total); err != nil {
		return err
	}
	tax := in.TaxAmountMinor
	if tax == 0 {
		tax = MoneyUnset
	}
	if err := PutMoney5(&payload, tax); err != nil {
		return err
	}
	PutByte(&payload, in.VATCode)
	PutByte(&payload, in.Department)
	PutByte(&payload, in.PaymentMethod)
	PutByte(&payload, in.PaymentSubject)
	if err := PutString1251(&payload, in.GoodName, 128, false); err != nil {
		return err
	}

	rsp, err := c.tr.Exchange(CmdOperationV2, payload)
	if err != nil {
		return err
	}
	if _, err := extractError(CmdOperationV2, rsp); err != nil {
		return err
	}
	return nil
}

// CloseReceiptV2Input - параметры команды 0xFF45.
//
// Используются только два типа оплаты: cash (Сумма наличных) и cashless (Сумма типа
// оплаты 2, "БЕЗНАЛИЧНЫМИ"). Остальные поля заполняются нулями.
type CloseReceiptV2Input struct {
	CashMinor     int64  // Сумма наличных
	CashlessMinor int64  // Сумма типа оплаты 2 (безналичные)
	RoundingMinor int64  // Округление до рубля в копейках (1 байт, 0..99)
	TaxSystemBit  byte   // СНО, бит из FiscalConfig
	Text          string // 0..64 символа
}

// CloseReceiptV2Result - распарсенный ответ FF45h.
type CloseReceiptV2Result struct {
	ChangeMinor int64     // сдача
	FDNumber    uint32    // номер ФД
	FiscalSign  uint32    // фискальный признак
	DateTime    time.Time // если ККТ настроена на расширенный ответ
	HasDateTime bool
	Raw         []byte
}

// CloseReceiptV2 - команда 0xFF45 "Закрытие чека расширенное вариант №2".
//
// В этой команде передаётся 16 типов оплаты по 5 байт. Налоги не передаются
// (значение 0xFFFFFFFFFF трактуется кассой как "посчитай сам") - это работает в
// режиме начисления налогов 0/2/3 (см. примечание к команде в спецификации).
func (c *Client) CloseReceiptV2(ctx context.Context, in CloseReceiptV2Input) (*CloseReceiptV2Result, error) {
	_ = ctx
	if in.RoundingMinor < 0 || in.RoundingMinor > 99 {
		return nil, fmt.Errorf("RoundingMinor=%d должно быть 0..99 коп", in.RoundingMinor)
	}
	if in.TaxSystemBit == 0 {
		return nil, fmt.Errorf("TaxSystemBit=0")
	}

	var payload []byte
	PutPassword(&payload, c.sysadminPassword)
	if err := PutMoney5(&payload, in.CashMinor); err != nil {
		return nil, err
	}
	if err := PutMoney5(&payload, in.CashlessMinor); err != nil {
		return nil, err
	}
	for i := 0; i < 14; i++ {
		if err := PutMoney5(&payload, 0); err != nil {
			return nil, err
		}
	}
	PutByte(&payload, byte(in.RoundingMinor))
	for i := 0; i < 6; i++ {
		if err := PutMoney5(&payload, MoneyUnset); err != nil {
			return nil, err
		}
	}
	PutByte(&payload, in.TaxSystemBit)
	if in.Text != "" {
		if err := PutString1251(&payload, in.Text, 64, false); err != nil {
			return nil, err
		}
	}

	rsp, err := c.tr.Exchange(CmdCloseReceipt, payload)
	if err != nil {
		return nil, err
	}
	tail, err := extractError(CmdCloseReceipt, rsp)
	if err != nil {
		return nil, err
	}

	if len(tail) < 13 {
		return nil, fmt.Errorf("ответ %s слишком короткий: %d байт", CmdCloseReceipt.Hex(), len(tail))
	}
	change, err := ReadMoney5(tail[0:5])
	if err != nil {
		return nil, err
	}
	fd, err := ReadUint32LE(tail[5:9])
	if err != nil {
		return nil, err
	}
	fs, err := ReadUint32LE(tail[9:13])
	if err != nil {
		return nil, err
	}
	res := &CloseReceiptV2Result{
		ChangeMinor: change,
		FDNumber:    fd,
		FiscalSign:  fs,
		Raw:         append([]byte(nil), tail...),
	}
	if len(tail) >= 18 {
		if dt, err := ReadDateTime5(tail[13:18]); err == nil {
			res.DateTime = dt
			res.HasDateTime = true
		}
	}
	return res, nil
}
