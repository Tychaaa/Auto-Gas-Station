// Пакет fiscal - адаптеры фискализации для бэкенда. Содержит абстрактный
// интерфейс Adapter и его реализацию поверх ККТ PayOnline-01-ФА (подпакет kkt).
package fiscal

import (
	"context"
	"errors"
	"time"
)

// PaymentKind - нормализованный тип оплаты для команды закрытия чека (FF45h).
type PaymentKind string

const (
	PaymentCash     PaymentKind = "cash"
	PaymentCashless PaymentKind = "cashless"
)

// ReceiptInput — доменное описание позиции чека из модели транзакции перед кодированием в ККТ.
type ReceiptInput struct {
	// TransactionID - идентификатор транзакции для логов.
	TransactionID string

	// GoodName - наименование товара (например, "АИ-92"). До 128 байт WIN1251.
	GoodName string

	// QuantityMicro - количество литров с 6 знаками после запятой (литры * 1_000_000).
	QuantityMicro int64

	// UnitPriceMinor - цена за литр в копейках.
	UnitPriceMinor int64

	// TotalMinor - итог по позиции в копейках.
	TotalMinor int64

	// PaymentKind - "cash" или "cashless".
	PaymentKind PaymentKind

	// RoundingMinor - копейки округления до рубля (0..99). Обычно 0.
	RoundingMinor int64
}

// Validate выполняет минимальную проверку входных данных. Полная валидация
// (диапазоны, маппинги) делается уже внутри адаптера.
func (r ReceiptInput) Validate() error {
	if r.GoodName == "" {
		return errors.New("receipt good name is empty")
	}
	if r.QuantityMicro <= 0 {
		return errors.New("receipt quantity must be > 0")
	}
	if r.UnitPriceMinor <= 0 {
		return errors.New("receipt unit price must be > 0")
	}
	if r.TotalMinor <= 0 {
		return errors.New("receipt total must be > 0")
	}
	switch r.PaymentKind {
	case PaymentCash, PaymentCashless:
	default:
		return errors.New("receipt payment kind must be cash or cashless")
	}
	if r.RoundingMinor < 0 || r.RoundingMinor > 99 {
		return errors.New("rounding must be in 0..99")
	}
	return nil
}

// Result - результат успешной фискализации.
type Result struct {
	// FDNumber - номер фискального документа (тег 1040).
	FDNumber uint32

	// FiscalSign - фискальный признак документа (тег 1077).
	FiscalSign uint32

	// ChangeMinor - сдача (для безнала всегда 0).
	ChangeMinor int64

	// ShiftNumber - номер открытой смены.
	ShiftNumber uint16

	// ReceiptNumber - номер чека внутри смены.
	ReceiptNumber uint16

	// HasDateTime/DateTime - время чека, если ККТ настроена на расширенный ответ.
	HasDateTime bool
	DateTime    time.Time
}

// Категория ошибки. Сервис фискализации использует её, чтобы решить, как
// обработать ошибку (откатить транзакцию, попросить открыть смену и т.д.).
type ErrorKind int

const (
	// ErrKindUnknown - не удалось классифицировать.
	ErrKindUnknown ErrorKind = iota
	// ErrKindNoLink - не удалось подключиться или потеряно соединение с ККТ.
	ErrKindNoLink
	// ErrKindShiftClosed - смена закрыта или просрочена, нужна ручная переоткрытка.
	ErrKindShiftClosed
	// ErrKindOperationFailed - ККТ отвергла команду FF46h (Операция V2).
	ErrKindOperationFailed
	// ErrKindCloseFailed - ККТ отвергла команду FF45h (закрытие чека).
	ErrKindCloseFailed
	// ErrKindBadInput - входные данные не прошли валидацию (мы не должны были этого допустить).
	ErrKindBadInput
)

// Error - ошибка адаптера с категорией.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// WrapError создаёт ошибку нужной категории.
func WrapError(kind ErrorKind, err error) *Error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

// Adapter - интерфейс провайдера фискализации.
// Реализация поверх ККТ PayOnline-01-ФА живёт в kkt_adapter.go.
type Adapter interface {
	// Fiscalize формирует и проводит чек на ККТ. Любая ошибка возвращается как *Error.
	Fiscalize(ctx context.Context, input ReceiptInput) (Result, error)
}
