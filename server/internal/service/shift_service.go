package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/fiscal"
	"AUTO-GAS-STATION/server/internal/model"
)

// KKTShiftStateRepository - интерфейс хранилища состояния смены.
type KKTShiftStateRepository interface {
	Load(ctx context.Context) (*model.KKTShiftState, error)
	Save(ctx context.Context, state model.KKTShiftState) error
	Clear(ctx context.Context) error
}

// HeaderLinesRepository - интерфейс хранилища заголовочных строк чека.
type HeaderLinesRepository interface {
	List(ctx context.Context) ([]model.HeaderLine, error)
	Replace(ctx context.Context, lines []model.HeaderLine) error
	Create(ctx context.Context, line model.HeaderLine) (model.HeaderLine, error)
	Update(ctx context.Context, line model.HeaderLine) error
	Delete(ctx context.Context, id int64) error
}

// ShiftServiceConfig - параметры ShiftService.
type ShiftServiceConfig struct {
	// AutoCloseAt - время ежедневного автозакрытия смены (формат "HH:MM").
	AutoCloseAt string
}

// ShiftStatusSnapshot - агрегированное состояние смены для API/UI.
type ShiftStatusSnapshot struct {
	IsOpen      bool
	IsExpired   bool
	ShiftNumber uint16
	ReceiptNum  uint16
	OpenedAt    *time.Time // nil если не отслеживается в SQLite
	HoursOpen   float64    // количество часов, на которое открыта смена
	HoursLeft   float64    // осталось часов до 24ч лимита (отрицательное = уже просрочена)
}

var autoCloseRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// ShiftService управляет посменным циклом ККТ:
// - авто-открытие/закрытие по расписанию
// - ручное закрытие из адм.панели
// - предоставление заголовков чека адаптеру (реализует fiscal.HeaderLinesProvider)
// - персистентность состояния смены (реализует fiscal.ShiftStateSink)
type ShiftService struct {
	adapter     fiscal.ShiftAdapter
	shiftRepo   KKTShiftStateRepository
	headerRepo  HeaderLinesRepository
	kiosk       *KioskService
	log         *slog.Logger
	autoCloseAt string // "HH:MM"

	cancel context.CancelFunc
	done   chan struct{}
}

// NewShiftService создаёт ShiftService. adapter может быть nil сразу после создания —
// вызовите SetAdapter перед первым использованием.
func NewShiftService(
	adapter fiscal.ShiftAdapter,
	shiftRepo KKTShiftStateRepository,
	headerRepo HeaderLinesRepository,
	kiosk *KioskService,
	log *slog.Logger,
	cfg ShiftServiceConfig,
) *ShiftService {
	autoCloseAt := cfg.AutoCloseAt
	if !autoCloseRe.MatchString(autoCloseAt) {
		log.Warn("shift_service.invalid_auto_close_at", slog.String("value", autoCloseAt), slog.String("fallback", "00:00"))
		autoCloseAt = "00:00"
	}
	return &ShiftService{
		adapter:     adapter,
		shiftRepo:   shiftRepo,
		headerRepo:  headerRepo,
		kiosk:       kiosk,
		log:         log.With(slog.String("component", "shift_service")),
		autoCloseAt: autoCloseAt,
	}
}

// SetAdapter устанавливает адаптер ККТ. Вызывается один раз из app.go после создания KKTAdapter.
func (s *ShiftService) SetAdapter(adapter fiscal.ShiftAdapter) {
	s.adapter = adapter
}

// --- fiscal.HeaderLinesProvider ---

// RenderHeaderLines возвращает тексты строк-заголовков, упорядоченных по position.
func (s *ShiftService) RenderHeaderLines(ctx context.Context) ([]string, error) {
	lines, err := s.headerRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	texts := make([]string, 0, len(lines))
	for _, l := range lines {
		texts = append(texts, l.Text)
	}
	return texts, nil
}

// --- fiscal.ShiftStateSink ---

// LoadShiftState возвращает сохранённое состояние смены или nil, если запись отсутствует.
func (s *ShiftService) LoadShiftState(ctx context.Context) (*fiscal.ShiftState, error) {
	state, err := s.shiftRepo.Load(ctx)
	if err != nil || state == nil {
		return nil, err
	}
	return &fiscal.ShiftState{ShiftNumber: state.ShiftNumber, OpenedAt: state.OpenedAt}, nil
}

// SaveShiftOpened сохраняет информацию об открытой смене.
func (s *ShiftService) SaveShiftOpened(ctx context.Context, shiftNumber uint16, openedAt time.Time) error {
	return s.shiftRepo.Save(ctx, model.KKTShiftState{ShiftNumber: shiftNumber, OpenedAt: openedAt})
}

// ClearShiftState удаляет запись состояния смены (после закрытия).
func (s *ShiftService) ClearShiftState(ctx context.Context) error {
	return s.shiftRepo.Clear(ctx)
}

// --- Операции администратора ---

// Status возвращает агрегированный снимок состояния смены.
func (s *ShiftService) Status(ctx context.Context) (ShiftStatusSnapshot, error) {
	if s.adapter == nil {
		return ShiftStatusSnapshot{}, fmt.Errorf("shift adapter not initialized")
	}
	kktStatus, err := s.adapter.ShiftStatus(ctx)
	if err != nil {
		return ShiftStatusSnapshot{}, err
	}
	snap := ShiftStatusSnapshot{
		IsOpen:      kktStatus.IsOpen,
		IsExpired:   kktStatus.IsExpired,
		ShiftNumber: kktStatus.ShiftNumber,
		ReceiptNum:  kktStatus.ReceiptNum,
	}
	if state, _ := s.shiftRepo.Load(ctx); state != nil {
		t := state.OpenedAt
		snap.OpenedAt = &t
		snap.HoursOpen = time.Since(state.OpenedAt).Hours()
		snap.HoursLeft = 24 - snap.HoursOpen
	}
	return snap, nil
}

// CloseNow закрывает смену Z-отчётом немедленно.
// Устанавливает kiosk maintenance на время закрытия.
func (s *ShiftService) CloseNow(ctx context.Context) (fiscal.ZReportResult, error) {
	if s.adapter == nil {
		return fiscal.ZReportResult{}, fmt.Errorf("shift adapter not initialized")
	}
	if s.kiosk != nil {
		s.kiosk.SetMaintenance(true, KioskReasonShiftClosing)
		defer func() {
			if s.kiosk != nil {
				s.kiosk.ClearMaintenanceIfReason(KioskReasonShiftClosing)
			}
		}()
	}

	result, err := s.adapter.CloseShiftZ(ctx)
	if err != nil {
		return fiscal.ZReportResult{}, err
	}
	if clearErr := s.shiftRepo.Clear(ctx); clearErr != nil {
		s.log.Warn("shift_service.clear_state_failed", slog.Any("err", clearErr))
	}
	s.log.Info("shift_service.closed",
		slog.Int("shift_number", int(result.ShiftNumber)),
		slog.Uint64("fd_number", uint64(result.FDNumber)),
		slog.Uint64("fiscal_sign", uint64(result.FiscalSign)),
	)
	return result, nil
}

// CalcStatusReport запрашивает отчёт о состоянии расчётов у ККТ.
func (s *ShiftService) CalcStatusReport(ctx context.Context) (fiscal.CalcStatusResult, error) {
	if s.adapter == nil {
		return fiscal.CalcStatusResult{}, fmt.Errorf("shift adapter not initialized")
	}
	return s.adapter.CalcStatusReport(ctx)
}

// --- Заголовочные строки (CRUD) ---

func (s *ShiftService) ListHeaderLines(ctx context.Context) ([]model.HeaderLine, error) {
	return s.headerRepo.List(ctx)
}

func (s *ShiftService) ReplaceHeaderLines(ctx context.Context, lines []model.HeaderLine) error {
	return s.headerRepo.Replace(ctx, lines)
}

func (s *ShiftService) CreateHeaderLine(ctx context.Context, line model.HeaderLine) (model.HeaderLine, error) {
	return s.headerRepo.Create(ctx, line)
}

func (s *ShiftService) UpdateHeaderLine(ctx context.Context, line model.HeaderLine) error {
	return s.headerRepo.Update(ctx, line)
}

func (s *ShiftService) DeleteHeaderLine(ctx context.Context, id int64) error {
	return s.headerRepo.Delete(ctx, id)
}

// --- Фоновое автозакрытие ---

// StartAutoClose запускает фоновую горутину автозакрытия смены по расписанию.
func (s *ShiftService) StartAutoClose() {
	if s.cancel != nil {
		return // уже запущен
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.autoCloseLoop(ctx)
}

// Stop корректно завершает фоновую горутину.
func (s *ShiftService) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
	s.cancel = nil
	s.done = nil
}

func (s *ShiftService) autoCloseLoop(ctx context.Context) {
	defer close(s.done)
	s.log.Info("shift_service.auto_close_loop_started", slog.String("auto_close_at", s.autoCloseAt))

	for {
		next := nextOccurrence(time.Now(), s.autoCloseAt)
		s.log.Info("shift_service.next_auto_close", slog.Time("at", next))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			s.log.Info("shift_service.auto_close_loop_stopped")
			return
		case <-timer.C:
		}

		s.log.Info("shift_service.auto_close_triggered")
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		if _, err := s.CloseNow(closeCtx); err != nil {
			s.log.Error("shift_service.auto_close_failed", slog.Any("err", err))
		}
		cancel()
	}
}

// nextOccurrence вычисляет ближайший момент после now, соответствующий времени "HH:MM".
func nextOccurrence(now time.Time, hhmm string) time.Time {
	var hh, mm int
	fmt.Sscanf(hhmm, "%d:%d", &hh, &mm) //nolint:errcheck
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}
