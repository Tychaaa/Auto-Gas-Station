package fiscal

import (
	"context"
	"time"
)

// ShiftState - снимок состояния смены, хранимый в SQLite (открыта ли, и когда).
type ShiftState struct {
	ShiftNumber uint16
	OpenedAt    time.Time
}

// ShiftOpenResult - результат открытия смены (команда 0xE0).
type ShiftOpenResult struct {
	ShiftNumber uint16
	FDNumber    uint32
	FiscalSign  uint32
}

// ZReportResult - результат Z-отчёта / закрытия смены (команда 0x41).
type ZReportResult struct {
	ShiftNumber uint16
	FDNumber    uint32
	FiscalSign  uint32
}

// ShiftStatusResult - текущее состояние смены, опрошенное у ККТ.
type ShiftStatusResult struct {
	IsOpen      bool
	IsExpired   bool
	ShiftNumber uint16
	ReceiptNum  uint16
}

// CalcStatusResult - результат отчёта о состоянии расчётов (0xFF37/0xFF38).
type CalcStatusResult struct {
	FDNumber             uint32
	FiscalSign           uint32
	UnconfirmedCount     uint32
	FirstUnconfirmedDate *time.Time
	HasDateTime          bool
	DateTime             time.Time
}

// ShiftAdapter - интерфейс для управления сменой ККТ; реализуется *KKTAdapter.
type ShiftAdapter interface {
	OpenShift(ctx context.Context) (ShiftOpenResult, error)
	CloseShiftZ(ctx context.Context) (ZReportResult, error)
	ShiftStatus(ctx context.Context) (ShiftStatusResult, error)
	PrintLines(ctx context.Context, text string) error
	CalcStatusReport(ctx context.Context) (CalcStatusResult, error)
}

// HeaderLinesProvider - поставщик строк-заголовков для печати перед каждым чеком.
// Реализуется *service.ShiftService.
type HeaderLinesProvider interface {
	RenderHeaderLines(ctx context.Context) ([]string, error)
}

// ShiftStateSink - хранилище событий смены для персистентности через рестарты.
// Реализуется *service.ShiftService.
type ShiftStateSink interface {
	LoadShiftState(ctx context.Context) (*ShiftState, error)
	SaveShiftOpened(ctx context.Context, shiftNumber uint16, openedAt time.Time) error
	ClearShiftState(ctx context.Context) error
}
