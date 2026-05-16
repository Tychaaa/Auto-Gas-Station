package config

import (
	"fmt"

	"AUTO-GAS-STATION/server/internal/adapter/fiscal"
)

// Дефолты по документации PayOnline-01-ФА для типового подключения.
const (
	defaultKKTHost              = "192.168.1.33"
	defaultKKTPort              = 7778
	defaultKKTConnectTimeoutMs  = 5000
	defaultKKTReadTimeoutMs     = 30000
	defaultKKTByteTimeoutMs     = 200
	defaultKKTAckTimeoutMs      = 1000
	defaultKKTSysadminPassword  = uint32(30)
	defaultKKTOperatorPassword  = uint32(30)
	defaultKKTTaxSystem         = "USN_INCOME_EXPENSE"
	defaultKKTVATRate           = "NO_VAT"
	defaultKKTPaymentMethodSign = 4
	defaultKKTPaymentSubject    = 1
	defaultKKTShiftMaxHours     = 23
	defaultKKTAutoCloseAt       = "00:00"
)

func loadFiscalKKTFromEnv() (fiscal.Config, error) {
	cfg := fiscal.Config{
		Host:               envString("KKT_HOST", defaultKKTHost),
		Port:               envInt("KKT_PORT", defaultKKTPort),
		ConnectTimeoutMs:   envInt("KKT_CONNECT_TIMEOUT_MS", defaultKKTConnectTimeoutMs),
		ReadTimeoutMs:      envInt("KKT_READ_TIMEOUT_MS", defaultKKTReadTimeoutMs),
		ByteTimeoutMs:      envInt("KKT_BYTE_TIMEOUT_MS", defaultKKTByteTimeoutMs),
		AckTimeoutMs:       envInt("KKT_ACK_TIMEOUT_MS", defaultKKTAckTimeoutMs),
		SysadminPassword:   envUint32("KKT_SYSADMIN_PASSWORD", defaultKKTSysadminPassword),
		OperatorPassword:   envUint32("KKT_OPERATOR_PASSWORD", defaultKKTOperatorPassword),
		TaxSystem:          envString("KKT_TAX_SYSTEM", defaultKKTTaxSystem),
		VATRate:            envString("KKT_VAT_RATE", defaultKKTVATRate),
		Department:         envInt("KKT_DEPARTMENT", 0),
		PaymentMethodSign:  envInt("KKT_PAYMENT_METHOD_SIGN", defaultKKTPaymentMethodSign),
		PaymentSubjectSign: envInt("KKT_PAYMENT_SUBJECT_SIGN", defaultKKTPaymentSubject),
		DumpHex:            envBool("KKT_DUMP_HEX", false),
		ShiftMaxHours:      envInt("KKT_SHIFT_MAX_HOURS", defaultKKTShiftMaxHours),
		AutoCloseAt:        envString("KKT_AUTO_CLOSE_AT", defaultKKTAutoCloseAt),
	}
	if err := cfg.Validate(); err != nil {
		return fiscal.Config{}, fmt.Errorf("invalid KKT config: %w", err)
	}
	return cfg, nil
}
