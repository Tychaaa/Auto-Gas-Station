package model

import "time"

// KKTShiftReport - запись об успешном закрытии смены ККТ (Z-отчёт).
type KKTShiftReport struct {
	ID          int64
	ShiftNumber uint16
	FDNumber    uint32
	FiscalSign  uint32
	ClosedAt    time.Time
}

// KKTCalcReport - запись об отчёте о состоянии расчётов (FF37/FF38).
type KKTCalcReport struct {
	ID                   int64
	FDNumber             uint32
	FiscalSign           uint32
	UnconfirmedCount     uint32
	FirstUnconfirmedDate *time.Time
	KKTDateTime          *time.Time
	CreatedAt            time.Time
}
