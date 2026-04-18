package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type AZTTransport interface {
	Exchange(ctx context.Context, frame []byte) ([]byte, error)
	Close() error
}

type WindowsSerialTransport struct {
	file *os.File
}

func NewWindowsSerialTransport(cfg FuelSerialConfig) (*WindowsSerialTransport, error) {
	if cfg.Port == "" {
		return nil, fmt.Errorf("serial port is required")
	}

	path := normalizeWindowsCOMPort(cfg.Port)
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open serial port %s: %w", cfg.Port, err)
	}

	return &WindowsSerialTransport{
		file: file,
	}, nil
}

func (t *WindowsSerialTransport) Exchange(ctx context.Context, frame []byte) ([]byte, error) {
	if t == nil || t.file == nil {
		return nil, fmt.Errorf("serial transport is not initialized")
	}

	if err := t.file.SetDeadline(time.Now().Add(400 * time.Millisecond)); err != nil {
		return nil, fmt.Errorf("set serial deadline: %w", err)
	}

	if _, err := t.file.Write(frame); err != nil {
		return nil, fmt.Errorf("write serial frame: %w", err)
	}

	buffer := make([]byte, 256)
	readBuf := make([]byte, 0, 64)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, err := t.file.Read(buffer)
		if n > 0 {
			readBuf = append(readBuf, buffer[:n]...)
			if isAZTPacketComplete(readBuf) {
				return append([]byte(nil), readBuf...), nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("read serial frame: %w", err)
		}
	}
}

func (t *WindowsSerialTransport) Close() error {
	if t == nil || t.file == nil {
		return nil
	}
	return t.file.Close()
}

func normalizeWindowsCOMPort(port string) string {
	trimmed := strings.TrimSpace(port)
	if strings.HasPrefix(trimmed, `\\.\`) {
		return trimmed
	}
	return `\\.\` + trimmed
}
