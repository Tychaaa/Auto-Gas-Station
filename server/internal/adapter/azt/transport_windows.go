package azt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial"
)

type SerialConfig struct {
	Port     string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
	Address  int
}

type Transport interface {
	Exchange(ctx context.Context, frame []byte) ([]byte, error)
	Close() error
}

type WindowsSerialTransport struct {
	port serial.Port
	cfg  SerialConfig
}

const defaultSerialExchangeTimeout = 1200 * time.Millisecond

func NewWindowsSerialTransport(cfg SerialConfig) (*WindowsSerialTransport, error) {
	if cfg.Port == "" {
		return nil, fmt.Errorf("serial port is required")
	}

	mode, err := buildSerialMode(cfg)
	if err != nil {
		return nil, err
	}

	port, err := serial.Open(cfg.Port, mode)
	if err != nil {
		return nil, fmt.Errorf("open serial port %s: %w", cfg.Port, err)
	}

	if err := port.SetReadTimeout(defaultSerialExchangeTimeout); err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("set serial read timeout: %w", err)
	}

	_ = port.ResetInputBuffer()
	_ = port.ResetOutputBuffer()

	return &WindowsSerialTransport{
		port: port,
		cfg:  cfg,
	}, nil
}

func (t *WindowsSerialTransport) Exchange(ctx context.Context, frame []byte) ([]byte, error) {
	if t == nil || t.port == nil {
		return nil, fmt.Errorf("serial transport is not initialized")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	_ = t.port.ResetInputBuffer()

	if _, err := t.port.Write(frame); err != nil {
		return nil, fmt.Errorf("write serial frame: %w", err)
	}

	buffer := make([]byte, 256)
	readBuf := make([]byte, 0, 64)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		n, err := t.port.Read(buffer)
		if n > 0 {
			readBuf = append(readBuf, buffer[:n]...)
			if isPacketComplete(readBuf) {
				return append([]byte(nil), readBuf...), nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("read serial frame: %w", err)
		}
		if n == 0 {
			if len(readBuf) == 0 {
				return nil, fmt.Errorf("read serial frame timeout after %s", defaultSerialExchangeTimeout)
			}
			return nil, fmt.Errorf("incomplete serial frame (%d bytes) after %s", len(readBuf), defaultSerialExchangeTimeout)
		}
	}
}

func (t *WindowsSerialTransport) Close() error {
	if t == nil || t.port == nil {
		return nil
	}
	err := t.port.Close()
	t.port = nil
	return err
}

func buildSerialMode(cfg SerialConfig) (*serial.Mode, error) {
	mode := &serial.Mode{
		BaudRate: cfg.Baud,
	}

	switch cfg.DataBits {
	case 0:
		mode.DataBits = 8
	case 5, 6, 7, 8:
		mode.DataBits = cfg.DataBits
	default:
		return nil, fmt.Errorf("unsupported data bits: %d", cfg.DataBits)
	}

	switch cfg.StopBits {
	case 0, 1:
		mode.StopBits = serial.OneStopBit
	case 2:
		mode.StopBits = serial.TwoStopBits
	default:
		return nil, fmt.Errorf("unsupported stop bits: %d", cfg.StopBits)
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Parity)) {
	case "", "none", "n":
		mode.Parity = serial.NoParity
	case "even", "e":
		mode.Parity = serial.EvenParity
	case "odd", "o":
		mode.Parity = serial.OddParity
	case "mark", "m":
		mode.Parity = serial.MarkParity
	case "space", "s":
		mode.Parity = serial.SpaceParity
	default:
		return nil, fmt.Errorf("unsupported parity: %q", cfg.Parity)
	}

	if mode.BaudRate <= 0 {
		mode.BaudRate = 4800
	}

	return mode, nil
}
