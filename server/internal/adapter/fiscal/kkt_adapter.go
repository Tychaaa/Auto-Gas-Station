package fiscal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/fiscal/kkt"
)

// KKTAdapterOptions - параметры создания адаптера на базе ККТ.
type KKTAdapterOptions struct {
	Config              Config
	Logger              *slog.Logger
	HeaderLinesProvider HeaderLinesProvider // optional; nil = заголовки не печатаются
	ShiftStateSink      ShiftStateSink      // optional; nil = состояние смены не персистируется
}

// KKTAdapter - реализация Adapter поверх ККТ PayOnline-01-ФА.
type KKTAdapter struct {
	cfg  Config
	log  *slog.Logger
	hlp  HeaderLinesProvider
	sink ShiftStateSink
}

// NewKKTAdapter создаёт новый адаптер. Реальное соединение с ККТ открывается на
// каждом вызове и сразу после обмена закрывается.
func NewKKTAdapter(opts KKTAdapterOptions) (*KKTAdapter, error) {
	if opts.Logger == nil {
		return nil, errors.New("fiscal: KKT adapter requires logger")
	}
	if err := opts.Config.Validate(); err != nil {
		return nil, fmt.Errorf("fiscal: invalid KKT config: %w", err)
	}
	return &KKTAdapter{
		cfg:  opts.Config,
		log:  opts.Logger.With(slog.String("component", "fiscal_kkt")),
		hlp:  opts.HeaderLinesProvider,
		sink: opts.ShiftStateSink,
	}, nil
}

// dialClient открывает TCP-соединение с ККТ и создаёт Client.
// Вызывающий обязан вызвать tr.Close().
func (a *KKTAdapter) dialClient(ctx context.Context, log *slog.Logger) (*kkt.Transport, *kkt.Client, error) {
	tr, err := kkt.Dial(ctx, kkt.TransportOptions{
		Address:        a.cfg.Address(),
		ConnectTimeout: a.cfg.ConnectTimeout(),
		ReadTimeout:    a.cfg.ReadTimeout(),
		ByteTimeout:    a.cfg.ByteTimeout(),
		AckTimeout:     a.cfg.AckTimeout(),
		DumpHex:        a.cfg.DumpHex,
		Logger:         log,
	})
	if err != nil {
		return nil, nil, err
	}
	client := kkt.NewClient(kkt.ClientOptions{
		Transport:        tr,
		SysadminPassword: a.cfg.SysadminPassword,
		OperatorPassword: a.cfg.OperatorPassword,
		Logger:           log,
	})
	return tr, client, nil
}

// Fiscalize реализует Adapter. На каждом вызове открывает TCP, проверяет/открывает смену,
// печатает заголовок, отправляет FF46 и FF45, после чего закрывает соединение.
func (a *KKTAdapter) Fiscalize(ctx context.Context, input ReceiptInput) (Result, error) {
	if err := input.Validate(); err != nil {
		return Result{}, WrapError(ErrKindBadInput, err)
	}

	taxBit, err := a.cfg.TaxSystemBit()
	if err != nil {
		return Result{}, WrapError(ErrKindBadInput, err)
	}
	vatCode, err := a.cfg.VATCode()
	if err != nil {
		return Result{}, WrapError(ErrKindBadInput, err)
	}

	log := a.log.With(slog.String("transaction_id", input.TransactionID))

	tr, client, err := a.dialClient(ctx, log)
	if err != nil {
		return Result{}, WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	status, err := client.ShortStatus(ctx)
	if err != nil {
		return Result{}, WrapError(ErrKindNoLink, fmt.Errorf("ShortStatus: %w", err))
	}
	log.Info("kkt.status",
		slog.Int("mode", int(status.Mode)),
		slog.Int("submode", int(status.Submode)),
		slog.Bool("shift_open_flag", status.IsShiftOpen()),
		slog.Bool("receipt_open_flag", status.IsReceiptOpen()),
	)

	shift, err := client.ShiftParams(ctx)
	if err != nil {
		return Result{}, WrapError(ErrKindNoLink, fmt.Errorf("ShiftParams: %w", err))
	}
	log.Info("kkt.shift",
		slog.String("state", shift.StateName()),
		slog.Int("shift_number", int(shift.ShiftNumber)),
		slog.Int("receipt_number", int(shift.ReceiptNum)),
	)

	// Убеждаемся, что смена открыта (авто-открытие/перезапуск при необходимости).
	shift, err = a.ensureShiftOpen(ctx, client, log, shift)
	if err != nil {
		return Result{}, err
	}

	// Печатаем строки-заголовки чека (название АЗС, адрес и т.п.).
	a.printHeader(ctx, client, log)

	if err := client.OperationV2(ctx, kkt.OperationV2Input{
		OperationType:  kkt.OpSale,
		QuantityMicro:  input.QuantityMicro,
		UnitPriceMinor: input.UnitPriceMinor,
		TotalMinor:     input.TotalMinor,
		TaxAmountMinor: 0,
		VATCode:        vatCode,
		Department:     byte(a.cfg.Department),
		PaymentMethod:  byte(a.cfg.PaymentMethodSign),
		PaymentSubject: byte(a.cfg.PaymentSubjectSign),
		GoodName:       input.GoodName,
	}); err != nil {
		return Result{}, WrapError(ErrKindOperationFailed, fmt.Errorf("OperationV2: %w", err))
	}
	log.Info("kkt.operation_v2_ok")

	closeIn := kkt.CloseReceiptV2Input{
		RoundingMinor: input.RoundingMinor,
		TaxSystemBit:  taxBit,
	}
	switch input.PaymentKind {
	case PaymentCash:
		closeIn.CashMinor = input.TotalMinor
	case PaymentCashless:
		closeIn.CashlessMinor = input.TotalMinor
	default:
		return Result{}, WrapError(ErrKindBadInput, fmt.Errorf("unknown payment kind %q", input.PaymentKind))
	}

	closeRes, err := client.CloseReceiptV2(ctx, closeIn)
	if err != nil {
		return Result{}, WrapError(ErrKindCloseFailed, fmt.Errorf("CloseReceiptV2: %w", err))
	}
	log.Info("kkt.receipt_closed",
		slog.Uint64("fd_number", uint64(closeRes.FDNumber)),
		slog.Uint64("fiscal_sign", uint64(closeRes.FiscalSign)),
		slog.Int64("change_minor", closeRes.ChangeMinor),
		slog.Bool("has_datetime", closeRes.HasDateTime),
	)
	if closeRes.HasDateTime {
		log.Info("kkt.receipt_datetime", slog.Time("at", closeRes.DateTime))
	}

	return Result{
		FDNumber:      closeRes.FDNumber,
		FiscalSign:    closeRes.FiscalSign,
		ChangeMinor:   closeRes.ChangeMinor,
		ShiftNumber:   shift.ShiftNumber,
		ReceiptNumber: shift.ReceiptNum,
		HasDateTime:   closeRes.HasDateTime,
		DateTime:      closeRes.DateTime,
	}, nil
}

// ensureShiftOpen проверяет состояние смены и при необходимости открывает/перезапускает её.
// Возвращает актуальные ShiftParams (после возможного открытия).
func (a *KKTAdapter) ensureShiftOpen(ctx context.Context, client *kkt.Client, log *slog.Logger, shift *kkt.ShiftParams) (*kkt.ShiftParams, error) {
	needOpen := false

	switch {
	case shift.IsExpired():
		log.Info("kkt.shift_expired_before_fiscal")
		if err := a.doCloseShift(ctx, client, log); err != nil {
			return nil, WrapError(ErrKindOperationFailed, fmt.Errorf("CloseShiftZ (expired): %w", err))
		}
		needOpen = true

	case shift.IsOpen():
		// Проактивный перезапуск если смена близится к 24-часовому лимиту.
		if a.sink != nil && a.cfg.ShiftMaxHours > 0 {
			state, _ := a.sink.LoadShiftState(ctx)
			if state != nil && time.Since(state.OpenedAt) >= time.Duration(a.cfg.ShiftMaxHours)*time.Hour {
				log.Info("kkt.shift_approaching_limit",
					slog.Time("opened_at", state.OpenedAt),
					slog.Float64("hours_open", time.Since(state.OpenedAt).Hours()),
				)
				if err := a.doCloseShift(ctx, client, log); err != nil {
					return nil, WrapError(ErrKindOperationFailed, fmt.Errorf("CloseShiftZ (proactive): %w", err))
				}
				needOpen = true
			}
		}

	default:
		log.Info("kkt.shift_closed_before_fiscal")
		needOpen = true
	}

	if !needOpen {
		return shift, nil
	}

	openRes, err := client.OpenShift(ctx)
	if err != nil {
		return nil, WrapError(ErrKindShiftClosed, fmt.Errorf("OpenShift: %w", err))
	}
	log.Info("kkt.shift_open",
		slog.Int("operator", int(openRes.OperatorNumber)),
		slog.Uint64("fd_number", uint64(openRes.FDNumber)),
		slog.Uint64("fiscal_sign", uint64(openRes.FiscalSign)),
		slog.Bool("has_fiscal", openRes.HasFiscal),
	)

	// Запрашиваем актуальные параметры смены после открытия.
	newShift, err := client.ShiftParams(ctx)
	if err != nil {
		return nil, WrapError(ErrKindNoLink, fmt.Errorf("ShiftParams after open: %w", err))
	}
	log.Info("kkt.shift",
		slog.String("state", newShift.StateName()),
		slog.Int("shift_number", int(newShift.ShiftNumber)),
	)

	if a.sink != nil {
		if err := a.sink.SaveShiftOpened(ctx, newShift.ShiftNumber, time.Now()); err != nil {
			log.Warn("kkt.shift_state_save_failed", slog.Any("err", err))
		}
	}
	return newShift, nil
}

// doCloseShift закрывает смену Z-отчётом и очищает персистентное состояние.
func (a *KKTAdapter) doCloseShift(ctx context.Context, client *kkt.Client, log *slog.Logger) error {
	log.Info("kkt.shift_close_request")
	closeRes, err := client.CloseShiftZ(ctx)
	if err != nil {
		return err
	}
	log.Info("kkt.shift_closed",
		slog.Int("operator", int(closeRes.OperatorNumber)),
		slog.Uint64("fd_number", uint64(closeRes.FDNumber)),
		slog.Uint64("fiscal_sign", uint64(closeRes.FiscalSign)),
		slog.Bool("has_fiscal", closeRes.HasFiscal),
	)
	if a.sink != nil {
		if err := a.sink.ClearShiftState(ctx); err != nil {
			log.Warn("kkt.shift_state_clear_failed", slog.Any("err", err))
		}
	}
	return nil
}

// printHeader печатает строки заголовка перед чеком. Ошибки не фатальны.
func (a *KKTAdapter) printHeader(ctx context.Context, client *kkt.Client, log *slog.Logger) {
	if a.hlp == nil {
		return
	}
	lines, err := a.hlp.RenderHeaderLines(ctx)
	if err != nil {
		log.Warn("kkt.header_lines_error", slog.Any("err", err))
		return
	}
	if len(lines) == 0 {
		return
	}
	for _, line := range lines {
		if err := client.PrintString(ctx, line); err != nil {
			log.Warn("kkt.print_line_error", slog.String("text", line), slog.Any("err", err))
		}
	}
	log.Info("kkt.header_printed", slog.Int("lines", len(lines)))
}

// OpenShift открывает смену ККТ. Использует отдельное TCP-соединение.
func (a *KKTAdapter) OpenShift(ctx context.Context) (ShiftOpenResult, error) {
	log := a.log
	tr, client, err := a.dialClient(ctx, log)
	if err != nil {
		return ShiftOpenResult{}, WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	openRes, err := client.OpenShift(ctx)
	if err != nil {
		return ShiftOpenResult{}, WrapError(ErrKindOperationFailed, fmt.Errorf("OpenShift: %w", err))
	}
	log.Info("kkt.shift_open",
		slog.Int("operator", int(openRes.OperatorNumber)),
		slog.Uint64("fd_number", uint64(openRes.FDNumber)),
		slog.Uint64("fiscal_sign", uint64(openRes.FiscalSign)),
	)

	shift, err := client.ShiftParams(ctx)
	if err != nil {
		return ShiftOpenResult{}, WrapError(ErrKindNoLink, fmt.Errorf("ShiftParams after open: %w", err))
	}
	return ShiftOpenResult{
		ShiftNumber: shift.ShiftNumber,
		FDNumber:    openRes.FDNumber,
		FiscalSign:  openRes.FiscalSign,
	}, nil
}

// CloseShiftZ закрывает смену Z-отчётом. Использует отдельное TCP-соединение.
func (a *KKTAdapter) CloseShiftZ(ctx context.Context) (ZReportResult, error) {
	log := a.log
	tr, client, err := a.dialClient(ctx, log)
	if err != nil {
		return ZReportResult{}, WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	shift, err := client.ShiftParams(ctx)
	if err != nil {
		return ZReportResult{}, WrapError(ErrKindNoLink, fmt.Errorf("ShiftParams: %w", err))
	}
	log.Info("kkt.shift",
		slog.String("state", shift.StateName()),
		slog.Int("shift_number", int(shift.ShiftNumber)),
	)

	log.Info("kkt.shift_close_request", slog.Int("shift_number", int(shift.ShiftNumber)))
	closeRes, err := client.CloseShiftZ(ctx)
	if err != nil {
		return ZReportResult{}, WrapError(ErrKindOperationFailed, fmt.Errorf("CloseShiftZ: %w", err))
	}
	log.Info("kkt.shift_closed",
		slog.Int("operator", int(closeRes.OperatorNumber)),
		slog.Uint64("fd_number", uint64(closeRes.FDNumber)),
		slog.Uint64("fiscal_sign", uint64(closeRes.FiscalSign)),
	)
	return ZReportResult{
		ShiftNumber: shift.ShiftNumber,
		FDNumber:    closeRes.FDNumber,
		FiscalSign:  closeRes.FiscalSign,
	}, nil
}

// ShiftStatus возвращает текущее состояние смены (опрашивает ККТ).
func (a *KKTAdapter) ShiftStatus(ctx context.Context) (ShiftStatusResult, error) {
	log := a.log
	tr, client, err := a.dialClient(ctx, log)
	if err != nil {
		return ShiftStatusResult{}, WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	shift, err := client.ShiftParams(ctx)
	if err != nil {
		return ShiftStatusResult{}, WrapError(ErrKindNoLink, fmt.Errorf("ShiftParams: %w", err))
	}
	return ShiftStatusResult{
		IsOpen:      shift.IsOpen(),
		IsExpired:   shift.IsExpired(),
		ShiftNumber: shift.ShiftNumber,
		ReceiptNum:  shift.ReceiptNum,
	}, nil
}

// PrintLines печатает произвольный текст на чековой ленте вне чека.
func (a *KKTAdapter) PrintLines(ctx context.Context, text string) error {
	log := a.log
	tr, client, err := a.dialClient(ctx, log)
	if err != nil {
		return WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	if err := client.PrintLines(ctx, text); err != nil {
		return WrapError(ErrKindOperationFailed, fmt.Errorf("PrintLines: %w", err))
	}
	return nil
}

// CalcStatusReport запрашивает отчёт о состоянии расчётов (FF37 + FF38).
func (a *KKTAdapter) CalcStatusReport(ctx context.Context) (CalcStatusResult, error) {
	log := a.log
	tr, client, err := a.dialClient(ctx, log)
	if err != nil {
		return CalcStatusResult{}, WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	if err := client.ReportCalcStart(ctx); err != nil {
		return CalcStatusResult{}, WrapError(ErrKindOperationFailed, fmt.Errorf("ReportCalcStart: %w", err))
	}
	r, err := client.ReportCalcForm(ctx)
	if err != nil {
		return CalcStatusResult{}, WrapError(ErrKindOperationFailed, fmt.Errorf("ReportCalcForm: %w", err))
	}

	res := CalcStatusResult{
		FDNumber:         r.FDNumber,
		FiscalSign:       r.FiscalSign,
		UnconfirmedCount: r.UnconfirmedCount,
		HasDateTime:      r.HasDateTime,
		DateTime:         r.DateTime,
	}
	if r.HasFirstUnconfirmed {
		t := r.FirstUnconfirmedDate
		res.FirstUnconfirmedDate = &t
	}
	return res, nil
}
