package watchdog

import "context"

// Disabled — заглушка адаптера, возвращающая ErrWatchdogDisabled на любую
// операцию. Используется когда WATCHDOG_MODE=disabled или серийный порт
// явно не сконфигурирован, чтобы сервер можно было запустить без железа
// и без мока (например, на CI или при первичной разработке).
type Disabled struct{}

func NewDisabled() *Disabled { return &Disabled{} }

func (Disabled) Heartbeat(ctx context.Context) (Heartbeat, error) {
	return Heartbeat{}, ErrWatchdogDisabled
}

func (Disabled) Status(ctx context.Context) (Status, error) {
	return Status{}, ErrWatchdogDisabled
}

func (Disabled) RequestReset(ctx context.Context) error {
	return ErrWatchdogDisabled
}

func (Disabled) Close() error { return nil }
