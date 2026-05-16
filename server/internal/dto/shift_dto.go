package dto

// ShiftStatusResponse - ответ GET /admin/shift/status.
type ShiftStatusResponse struct {
	IsOpen      bool    `json:"is_open"`
	IsExpired   bool    `json:"is_expired"`
	ShiftNumber uint16  `json:"shift_number"`
	ReceiptNum  uint16  `json:"receipt_num"`
	OpenedAt    string  `json:"opened_at,omitempty"` // RFC3339, пустая если не отслеживается
	HoursOpen   float64 `json:"hours_open"`
	HoursLeft   float64 `json:"hours_left"`
}

// CloseShiftResponse - ответ POST /admin/shift/close.
type CloseShiftResponse struct {
	ShiftNumber uint16 `json:"shift_number"`
	FDNumber    uint32 `json:"fd_number"`
	FiscalSign  uint32 `json:"fiscal_sign"`
}

// CalcStatusResponse - ответ POST /admin/reports/calc-status.
type CalcStatusResponse struct {
	FDNumber             uint32 `json:"fd_number"`
	FiscalSign           uint32 `json:"fiscal_sign"`
	UnconfirmedCount     uint32 `json:"unconfirmed_count"`
	FirstUnconfirmedDate string `json:"first_unconfirmed_date,omitempty"` // "YYYY-MM-DD"
	DateTime             string `json:"datetime,omitempty"`               // RFC3339
}

// HeaderLineDTO - строка заголовка чека для API.
type HeaderLineDTO struct {
	ID       int64  `json:"id,omitempty"`
	Position int    `json:"position"`
	Text     string `json:"text"`
}

// ReplaceHeaderLinesRequest - тело PUT /admin/kkt/header-lines.
type ReplaceHeaderLinesRequest struct {
	Lines []HeaderLineDTO `json:"lines"`
}

// CreateHeaderLineRequest - тело POST /admin/kkt/header-lines.
type CreateHeaderLineRequest struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
}

// UpdateHeaderLineRequest - тело PUT /admin/kkt/header-lines/:id.
type UpdateHeaderLineRequest struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
}

// OpenShiftResponse - ответ POST /admin/shift/open.
type OpenShiftResponse struct {
	ShiftNumber uint16 `json:"shift_number"`
	FDNumber    uint32 `json:"fd_number"`
	FiscalSign  uint32 `json:"fiscal_sign"`
}

// ShiftReportDTO - запись из истории Z-отчётов.
type ShiftReportDTO struct {
	ID          int64  `json:"id"`
	ShiftNumber uint16 `json:"shift_number"`
	FDNumber    uint32 `json:"fd_number"`
	FiscalSign  uint32 `json:"fiscal_sign"`
	ClosedAt    string `json:"closed_at"` // RFC3339
}

// CalcReportDTO - запись из истории отчётов о состоянии расчётов.
type CalcReportDTO struct {
	ID                   int64  `json:"id"`
	FDNumber             uint32 `json:"fd_number"`
	FiscalSign           uint32 `json:"fiscal_sign"`
	UnconfirmedCount     uint32 `json:"unconfirmed_count"`
	FirstUnconfirmedDate string `json:"first_unconfirmed_date,omitempty"` // "YYYY-MM-DD"
	DateTime             string `json:"datetime,omitempty"`               // RFC3339
	CreatedAt            string `json:"created_at"`                       // RFC3339
}
