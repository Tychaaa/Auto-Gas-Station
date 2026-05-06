package fiscal

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Config - параметры подключения и реквизиты фискализации для PayOnline-01-ФА.
// Заполняется в config-слое из env. Здесь же лежат справочники СНО/НДС.
type Config struct {
	Host               string
	Port               int
	ConnectTimeoutMs   int
	ReadTimeoutMs      int
	ByteTimeoutMs      int
	AckTimeoutMs       int
	SysadminPassword   uint32
	OperatorPassword   uint32
	TaxSystem          string
	VATRate            string
	Department         int
	PaymentMethodSign  int
	PaymentSubjectSign int
	DumpHex            bool
}

// Address - "host:port" для TCP.
func (c Config) Address() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// ConnectTimeout - таймаут установки соединения.
func (c Config) ConnectTimeout() time.Duration {
	return time.Duration(c.ConnectTimeoutMs) * time.Millisecond
}

// ReadTimeout - таймаут ожидания ответа на команду.
func (c Config) ReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutMs) * time.Millisecond
}

// ByteTimeout - таймаут ожидания одного байта (ENQ/ACK/NAK).
func (c Config) ByteTimeout() time.Duration {
	return time.Duration(c.ByteTimeoutMs) * time.Millisecond
}

// AckTimeout - таймаут ожидания подтверждения после отправки кадра.
func (c Config) AckTimeout() time.Duration {
	return time.Duration(c.AckTimeoutMs) * time.Millisecond
}

// taxSystemBits - биты "Применяемая система налогообложения" для FF45h.
// ENVD не приводим (утратил силу с 2025), ESHN = ЕСХН.
var taxSystemBits = map[string]byte{
	"OSN":                1 << 0,
	"USN_INCOME":         1 << 1,
	"USN_INCOME_EXPENSE": 1 << 2,
	"ESHN":               1 << 4,
	"PSN":                1 << 5,
}

// vatRateCodes - коды налоговых ставок из таблицы FF46h.
var vatRateCodes = map[string]byte{
	"VAT_20":     0x01,
	"VAT_10":     0x02,
	"VAT_0":      0x04,
	"NO_VAT":     0x08,
	"VAT_20_120": 0x10,
	"VAT_10_110": 0x20,
	"VAT_5":      0x81,
	"VAT_7":      0x82,
	"VAT_5_105":  0x84,
	"VAT_7_107":  0x88,
}

// TaxSystemBit возвращает байт системы налогообложения.
func (c Config) TaxSystemBit() (byte, error) {
	b, ok := taxSystemBits[strings.ToUpper(strings.TrimSpace(c.TaxSystem))]
	if !ok {
		return 0, fmt.Errorf("unknown KKT tax system %q", c.TaxSystem)
	}
	return b, nil
}

// VATCode возвращает код ставки НДС.
func (c Config) VATCode() (byte, error) {
	v, ok := vatRateCodes[strings.ToUpper(strings.TrimSpace(c.VATRate))]
	if !ok {
		return 0, fmt.Errorf("unknown KKT VAT rate %q", c.VATRate)
	}
	return v, nil
}

// Validate проверяет согласованность конфига.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("KKT host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("KKT port=%d out of range 1..65535", c.Port)
	}
	if c.ConnectTimeoutMs <= 0 || c.ReadTimeoutMs <= 0 || c.ByteTimeoutMs <= 0 || c.AckTimeoutMs <= 0 {
		return errors.New("KKT timeouts must be > 0")
	}
	if _, err := c.TaxSystemBit(); err != nil {
		return err
	}
	if _, err := c.VATCode(); err != nil {
		return err
	}
	if c.Department < 0 || c.Department > 16 {
		return fmt.Errorf("KKT department=%d out of range 0..16", c.Department)
	}
	if c.PaymentMethodSign < 1 || c.PaymentMethodSign > 7 {
		return fmt.Errorf("KKT payment method sign=%d out of range 1..7", c.PaymentMethodSign)
	}
	if c.PaymentSubjectSign < 1 || c.PaymentSubjectSign > 30 {
		return fmt.Errorf("KKT payment subject sign=%d out of range 1..30", c.PaymentSubjectSign)
	}
	return nil
}
