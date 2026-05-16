package model

import "time"

// KKTShiftState - состояние текущей смены ККТ (хранится в SQLite, переживает рестарт).
type KKTShiftState struct {
	ShiftNumber uint16
	OpenedAt    time.Time
}

// HeaderLine - строка заголовка чека ККТ.
type HeaderLine struct {
	ID       int64
	Position int
	Text     string
}
