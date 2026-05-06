package fiscal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"AUTO-GAS-STATION/server/internal/adapter/fiscal/kkt"
)

// KKTAdapterOptions - параметры создания адаптера на базе ККТ.
type KKTAdapterOptions struct {
	Config Config
	Logger *slog.Logger
}

// KKTAdapter - реализация Adapter поверх ККТ PayOnline-01-ФА.
type KKTAdapter struct {
	cfg Config
	log *slog.Logger
}

// NewKKTAdapter создаёт новый адаптер. Реальное соединение с ККТ открывается на
// каждом вызове Fiscalize и сразу после обмена закрывается.
func NewKKTAdapter(opts KKTAdapterOptions) (*KKTAdapter, error) {
	if opts.Logger == nil {
		return nil, errors.New("fiscal: KKT adapter requires logger")
	}
	if err := opts.Config.Validate(); err != nil {
		return nil, fmt.Errorf("fiscal: invalid KKT config: %w", err)
	}
	return &KKTAdapter{cfg: opts.Config, log: opts.Logger.With(slog.String("component", "fiscal_kkt"))}, nil
}

// Fiscalize реализует Adapter. На каждом вызове открывает TCP, проверяет смену,
// отправляет FF46 и FF45, после чего закрывает соединение.
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
		return Result{}, WrapError(ErrKindNoLink, err)
	}
	defer tr.Close()

	client := kkt.NewClient(kkt.ClientOptions{
		Transport:        tr,
		SysadminPassword: a.cfg.SysadminPassword,
		OperatorPassword: a.cfg.OperatorPassword,
		Logger:           log,
	})

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
	if shift.IsExpired() {
		return Result{}, WrapError(ErrKindShiftClosed, errors.New("смена просрочена (>24ч), закройте и откройте её вручную"))
	}
	if !shift.IsOpen() {
		return Result{}, WrapError(ErrKindShiftClosed, errors.New("смена закрыта, откройте её вручную перед фискализацией"))
	}

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
