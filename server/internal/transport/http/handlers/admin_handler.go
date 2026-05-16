package handlers

import (
	"errors"
	nethttp "net/http"
	"strconv"
	"time"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	prices *service.PriceService
	txRepo service.TransactionRepository
	kiosk  *service.KioskService
	shift  *service.ShiftService
}

func NewAdminHandler(prices *service.PriceService, txRepo service.TransactionRepository, kiosk *service.KioskService, shift *service.ShiftService) *AdminHandler {
	return &AdminHandler{prices: prices, txRepo: txRepo, kiosk: kiosk, shift: shift}
}

func (h *AdminHandler) ListPriceVersions(c *gin.Context) {
	if h.prices == nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": "price service is not configured"})
		return
	}

	versions, err := h.prices.ListVersions(0)
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": versions})
}

func (h *AdminHandler) CreatePriceVersion(c *gin.Context) {
	if h.prices == nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": "price service is not configured"})
		return
	}

	var req dto.AdminCreatePriceVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	prices := make(map[string]float64, len(req.Items))
	for _, item := range req.Items {
		prices[item.FuelType] = item.PricePerLiter
	}

	version, err := h.prices.CreatePriceVersion(req.VersionTag, req.EffectiveFrom, prices)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.kiosk != nil {
		h.kiosk.ClearMaintenanceIfReason(service.KioskReasonNoPrices)
	}
	c.JSON(nethttp.StatusCreated, version)
}

func (h *AdminHandler) ListTransactions(c *gin.Context) {
	txs, err := h.txRepo.ListAll()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]dto.AdminTransactionView, 0, len(txs))
	for _, tx := range txs {
		items = append(items, toAdminTransactionView(tx))
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": items})
}

func (h *AdminHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	tx, err := h.txRepo.Get(id)
	if errors.Is(err, repository.ErrTransactionNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	events, err := h.txRepo.GetEvents(id)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, toAdminTransactionDetailsView(tx, events))
}

func toAdminTransactionView(tx *model.Transaction) dto.AdminTransactionView {
	return dto.AdminTransactionView{
		ID:            tx.ID,
		CreatedAt:     tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     tx.UpdatedAt.Format(time.RFC3339),
		FuelType:      tx.FuelType,
		OrderMode:     tx.OrderMode,
		Liters:        txDisplayLiters(tx),
		AmountRub:     txDisplayAmountRub(tx),
		Currency:      tx.Currency,
		Status:        string(tx.Status),
		PaymentStatus: string(tx.PaymentStatus),
		FiscalStatus:  string(tx.FiscalStatus),
		FuelingStatus: string(tx.FuelingStatus),
		ReceiptNumber: tx.ReceiptNumber,
		ErrorMessage:  txErrorMessage(tx),
	}
}

func toAdminTransactionDetailsView(tx *model.Transaction, events []model.TransactionEvent) dto.AdminTransactionDetailsView {
	eventDTOs := make([]dto.TransactionEventDTO, 0, len(events))
	for _, ev := range events {
		eventDTOs = append(eventDTOs, dto.TransactionEventDTO{
			EventType:  string(ev.EventType),
			OccurredAt: ev.OccurredAt.Format(time.RFC3339),
			Detail:     ev.Detail,
		})
	}

	return dto.AdminTransactionDetailsView{
		ID:                tx.ID,
		CreatedAt:         tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         tx.UpdatedAt.Format(time.RFC3339),
		FuelType:          tx.FuelType,
		OrderMode:         tx.OrderMode,
		AmountRub:         tx.AmountRub,
		Liters:            tx.Liters,
		Preset:            tx.Preset,
		PriceVersionTag:   tx.PriceVersionTag,
		UnitPriceRub:      float64(tx.UnitPriceMinor) / 100.0,
		ComputedAmountRub: float64(tx.ComputedAmountMinor) / 100.0,
		Currency:          tx.Currency,
		PricingSnapshotAt: formatOptionalTime(tx.PricingSnapshotAt),
		PriceLockedUntil:  formatOptionalTime(tx.PriceLockedUntil),
		PriceWasRepriced:  tx.PriceWasRepriced,
		Status:            string(tx.Status),
		PaymentStatus:     string(tx.PaymentStatus),
		FiscalStatus:      string(tx.FiscalStatus),
		FuelingStatus:     string(tx.FuelingStatus),
		PaymentProvider:   tx.PaymentProvider,
		PaymentSessionID:  tx.PaymentSessionID,
		PaymentError:      tx.PaymentError,
		ReceiptNumber:     tx.ReceiptNumber,
		FiscalError:       tx.FiscalError,
		FuelingSessionID:  tx.FuelingSessionID,
		DispensedLiters:   tx.DispensedLiters,
		DispenseComplete:  tx.DispenseComplete,
		DispensePartial:   tx.DispensePartial,
		FuelingError:      tx.FuelingError,
		AbandonReason:     tx.AbandonReason,
		Events:            eventDTOs,
	}
}

func txDisplayLiters(tx *model.Transaction) float64 {
	if tx.DispensedLiters > 0 {
		return tx.DispensedLiters
	}
	return tx.Liters
}

func txDisplayAmountRub(tx *model.Transaction) float64 {
	if tx.ComputedAmountMinor > 0 {
		return float64(tx.ComputedAmountMinor) / 100.0
	}
	return float64(tx.AmountRub)
}

func txErrorMessage(tx *model.Transaction) string {
	if tx.PaymentError != "" {
		return tx.PaymentError
	}
	if tx.FiscalError != "" {
		return tx.FiscalError
	}
	return tx.FuelingError
}

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ShiftStatus - GET /admin/shift/status
func (h *AdminHandler) ShiftStatus(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	snap, err := h.shift.Status(c.Request.Context())
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	resp := dto.ShiftStatusResponse{
		IsOpen:      snap.IsOpen,
		IsExpired:   snap.IsExpired,
		ShiftNumber: snap.ShiftNumber,
		ReceiptNum:  snap.ReceiptNum,
		HoursOpen:   snap.HoursOpen,
		HoursLeft:   snap.HoursLeft,
	}
	if snap.OpenedAt != nil {
		resp.OpenedAt = snap.OpenedAt.Format(time.RFC3339)
	}
	c.JSON(nethttp.StatusOK, resp)
}

// CloseShift - POST /admin/shift/close
func (h *AdminHandler) CloseShift(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	result, err := h.shift.CloseNow(c.Request.Context())
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, dto.CloseShiftResponse{
		ShiftNumber: result.ShiftNumber,
		FDNumber:    result.FDNumber,
		FiscalSign:  result.FiscalSign,
	})
}

// CalcStatusReport - POST /admin/reports/calc-status
func (h *AdminHandler) CalcStatusReport(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	result, err := h.shift.CalcStatusReport(c.Request.Context())
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	resp := dto.CalcStatusResponse{
		FDNumber:         result.FDNumber,
		FiscalSign:       result.FiscalSign,
		UnconfirmedCount: result.UnconfirmedCount,
	}
	if result.FirstUnconfirmedDate != nil {
		resp.FirstUnconfirmedDate = result.FirstUnconfirmedDate.Format("2006-01-02")
	}
	if result.HasDateTime {
		resp.DateTime = result.DateTime.Format(time.RFC3339)
	}
	c.JSON(nethttp.StatusOK, resp)
}

// ListHeaderLines - GET /admin/kkt/header-lines
func (h *AdminHandler) ListHeaderLines(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	lines, err := h.shift.ListHeaderLines(c.Request.Context())
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]dto.HeaderLineDTO, 0, len(lines))
	for _, l := range lines {
		items = append(items, dto.HeaderLineDTO{ID: l.ID, Position: l.Position, Text: l.Text})
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": items})
}

// ReplaceHeaderLines - PUT /admin/kkt/header-lines (bulk replace)
func (h *AdminHandler) ReplaceHeaderLines(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	var req dto.ReplaceHeaderLinesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	lines := make([]model.HeaderLine, 0, len(req.Lines))
	for i, l := range req.Lines {
		pos := l.Position
		if pos == 0 {
			pos = i + 1
		}
		lines = append(lines, model.HeaderLine{Position: pos, Text: l.Text})
	}
	if err := h.shift.ReplaceHeaderLines(c.Request.Context(), lines); err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"ok": true})
}

// CreateHeaderLine - POST /admin/kkt/header-lines
func (h *AdminHandler) CreateHeaderLine(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	var req dto.CreateHeaderLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	created, err := h.shift.CreateHeaderLine(c.Request.Context(), model.HeaderLine{
		Position: req.Position,
		Text:     req.Text,
	})
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, dto.HeaderLineDTO{ID: created.ID, Position: created.Position, Text: created.Text})
}

// UpdateHeaderLine - PUT /admin/kkt/header-lines/:id
func (h *AdminHandler) UpdateHeaderLine(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.UpdateHeaderLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	if err := h.shift.UpdateHeaderLine(c.Request.Context(), model.HeaderLine{
		ID:       id,
		Position: req.Position,
		Text:     req.Text,
	}); errors.Is(err, repository.ErrHeaderLineNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "header line not found"})
		return
	} else if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"ok": true})
}

// DeleteHeaderLine - DELETE /admin/kkt/header-lines/:id
func (h *AdminHandler) DeleteHeaderLine(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.shift.DeleteHeaderLine(c.Request.Context(), id); errors.Is(err, repository.ErrHeaderLineNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "header line not found"})
		return
	} else if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"ok": true})
}

// OpenShift - POST /admin/shift/open
func (h *AdminHandler) OpenShift(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	result, err := h.shift.OpenNow(c.Request.Context())
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, dto.OpenShiftResponse{
		ShiftNumber: result.ShiftNumber,
		FDNumber:    result.FDNumber,
		FiscalSign:  result.FiscalSign,
	})
}

// ListShiftReports - GET /admin/shift/reports
func (h *AdminHandler) ListShiftReports(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	limit, offset := parseLimitOffset(c, 200, 1000)
	reps, err := h.shift.ListShiftReports(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]dto.ShiftReportDTO, 0, len(reps))
	for _, r := range reps {
		items = append(items, dto.ShiftReportDTO{
			ID:          r.ID,
			ShiftNumber: r.ShiftNumber,
			FDNumber:    r.FDNumber,
			FiscalSign:  r.FiscalSign,
			ClosedAt:    r.ClosedAt.Format(time.RFC3339),
		})
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": items})
}

// DeleteShiftReport - DELETE /admin/shift/reports/:id
func (h *AdminHandler) DeleteShiftReport(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.shift.DeleteShiftReport(c.Request.Context(), id); errors.Is(err, repository.ErrKKTShiftReportNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "shift report not found"})
		return
	} else if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(nethttp.StatusNoContent)
}

// ListCalcReports - GET /admin/reports/calc-status/history
func (h *AdminHandler) ListCalcReports(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	limit, offset := parseLimitOffset(c, 200, 1000)
	reps, err := h.shift.ListCalcReports(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]dto.CalcReportDTO, 0, len(reps))
	for _, r := range reps {
		item := dto.CalcReportDTO{
			ID:               r.ID,
			FDNumber:         r.FDNumber,
			FiscalSign:       r.FiscalSign,
			UnconfirmedCount: r.UnconfirmedCount,
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		}
		if r.FirstUnconfirmedDate != nil {
			item.FirstUnconfirmedDate = r.FirstUnconfirmedDate.Format("2006-01-02")
		}
		if r.KKTDateTime != nil {
			item.DateTime = r.KKTDateTime.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": items})
}

// DeleteCalcReport - DELETE /admin/reports/calc-status/history/:id
func (h *AdminHandler) DeleteCalcReport(c *gin.Context) {
	if h.shift == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "shift service not available"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.shift.DeleteCalcReport(c.Request.Context(), id); errors.Is(err, repository.ErrKKTCalcReportNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "calc report not found"})
		return
	} else if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(nethttp.StatusNoContent)
}

func parseLimitOffset(c *gin.Context, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
		if limit > maxLimit {
			limit = maxLimit
		}
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v >= 0 {
		offset = v
	}
	return
}
