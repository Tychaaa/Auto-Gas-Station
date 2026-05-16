package kkt

import (
	"errors"
	"fmt"
	"time"
)

// ShiftOpenResult - разобранный ответ команды 0xE0 (Открыть смену).
type ShiftOpenResult struct {
	OperatorNumber byte
	FDNumber       uint32
	FiscalSign     uint32
	DateTime       time.Time
	HasFiscal      bool // true если касса вернула FD/FP/DT (расширенный ответ)
}

// ParseShiftOpenResult разбирает хвост ответа команды 0xE0 (после снятия кода ошибки).
//
// Базовый ответ: 1 байт (номер оператора).
// Расширенный ответ: +4 (FD) +4 (FP) +5 (DT) = 14 байт.
func ParseShiftOpenResult(tail []byte) (*ShiftOpenResult, error) {
	if len(tail) < 1 {
		return nil, errors.New("ответ OpenShift пустой после кода ошибки")
	}
	r := &ShiftOpenResult{OperatorNumber: tail[0]}
	if len(tail) >= 15 {
		fd, err := ReadUint32LE(tail[1:5])
		if err != nil {
			return nil, fmt.Errorf("OpenShift FD: %w", err)
		}
		fp, err := ReadUint32LE(tail[5:9])
		if err != nil {
			return nil, fmt.Errorf("OpenShift FP: %w", err)
		}
		dt, err := ReadDateTime5(tail[9:14])
		if err != nil {
			return nil, fmt.Errorf("OpenShift DT: %w", err)
		}
		r.FDNumber = fd
		r.FiscalSign = fp
		r.DateTime = dt
		r.HasFiscal = true
	}
	return r, nil
}

// ZReportResult - разобранный ответ команды 0x41 (суточный отчёт с гашением).
type ZReportResult struct {
	OperatorNumber byte
	FDNumber       uint32
	FiscalSign     uint32
	DateTime       time.Time
	HasFiscal      bool
}

// ParseZReportResult разбирает хвост ответа команды 0x41 (после снятия кода ошибки).
//
// Структура та же, что у OpenShift: 1 байт оператор + опционально FD+FP+DT.
func ParseZReportResult(tail []byte) (*ZReportResult, error) {
	if len(tail) < 1 {
		return nil, errors.New("ответ ZReport пустой после кода ошибки")
	}
	r := &ZReportResult{OperatorNumber: tail[0]}
	if len(tail) >= 15 {
		fd, err := ReadUint32LE(tail[1:5])
		if err != nil {
			return nil, fmt.Errorf("ZReport FD: %w", err)
		}
		fp, err := ReadUint32LE(tail[5:9])
		if err != nil {
			return nil, fmt.Errorf("ZReport FP: %w", err)
		}
		dt, err := ReadDateTime5(tail[9:14])
		if err != nil {
			return nil, fmt.Errorf("ZReport DT: %w", err)
		}
		r.FDNumber = fd
		r.FiscalSign = fp
		r.DateTime = dt
		r.HasFiscal = true
	}
	return r, nil
}

// CalcStatusReport - разобранный ответ команды FF38h
// (Сформировать отчёт о состоянии расчётов, ФФД тип 0x17).
type CalcStatusReport struct {
	FDNumber             uint32
	FiscalSign           uint32
	UnconfirmedCount     uint32
	FirstUnconfirmedDate time.Time // нулевая если UnconfirmedCount == 0
	HasFirstUnconfirmed  bool
	DateTime             time.Time
	HasDateTime          bool
}

// ParseCalcStatusReport разбирает хвост ответа FF38h (после снятия кода ошибки).
//
// Базовый ответ: FD(4) + FP(4) + UnconfirmedCount(4) + FirstUnconfirmedDate(3) = 15 байт.
// Расширенный: +DT(5) = 20 байт.
func ParseCalcStatusReport(tail []byte) (*CalcStatusReport, error) {
	if len(tail) < 15 {
		return nil, fmt.Errorf("ответ ReportCalcForm слишком короткий: %d байт (нужно 15)", len(tail))
	}
	fd, err := ReadUint32LE(tail[0:4])
	if err != nil {
		return nil, fmt.Errorf("CalcReport FD: %w", err)
	}
	fp, err := ReadUint32LE(tail[4:8])
	if err != nil {
		return nil, fmt.Errorf("CalcReport FP: %w", err)
	}
	cnt, err := ReadUint32LE(tail[8:12])
	if err != nil {
		return nil, fmt.Errorf("CalcReport UnconfirmedCount: %w", err)
	}

	r := &CalcStatusReport{
		FDNumber:         fd,
		FiscalSign:       fp,
		UnconfirmedCount: cnt,
	}

	if cnt > 0 {
		d, err := readDate3(tail[12:15])
		if err == nil {
			r.FirstUnconfirmedDate = d
			r.HasFirstUnconfirmed = true
		}
	}

	if len(tail) >= 20 {
		dt, err := ReadDateTime5(tail[15:20])
		if err == nil {
			r.DateTime = dt
			r.HasDateTime = true
		}
	}
	return r, nil
}

// readDate3 читает 3 байта YYMMDD и возвращает time.Time (только дата).
func readDate3(p []byte) (time.Time, error) {
	if len(p) < 3 {
		return time.Time{}, errors.New("readDate3: нужно 3 байта")
	}
	year := 2000 + int(p[0])
	month := int(p[1])
	day := int(p[2])
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("некорректная дата %02d-%02d-%02d", year, month, day)
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}
