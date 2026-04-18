package main

import (
	"context"
	"fmt"
)

type AZTStatusSnapshot struct {
	StatusCode      byte
	ReasonCode      byte
	ProviderStatus  string
	DispensedLiters float64
	Completed       bool
	Partial         bool
}

type AZTDeliveryTotals struct {
	DispensedLiters float64
	AmountRub       int64
	PriceRub        int64
}

type AZTMasterClient struct {
	transport AZTTransport
	startByte byte
	address   byte
}

func NewAZTMasterClient(transport AZTTransport, address int) (*AZTMasterClient, error) {
	if transport == nil {
		return nil, fmt.Errorf("azt transport is required")
	}
	if address < 1 || address > 15 {
		return nil, fmt.Errorf("azt address must be in range 1..15")
	}

	return &AZTMasterClient{
		transport: transport,
		startByte: aztSTX,
		address:   byte(0x20 + address),
	}, nil
}

func (c *AZTMasterClient) Close() error {
	if c == nil || c.transport == nil {
		return nil
	}
	return c.transport.Close()
}

func (c *AZTMasterClient) GetStatus(ctx context.Context) (AZTStatusSnapshot, error) {
	resp, err := c.exchange(ctx, aztCmdStatus, nil)
	if err != nil {
		return AZTStatusSnapshot{}, err
	}
	if resp.ShortResponse != nil {
		return AZTStatusSnapshot{}, fmt.Errorf("unexpected short response for status")
	}
	if len(resp.Data) < 1 {
		return AZTStatusSnapshot{}, fmt.Errorf("status response is empty")
	}

	snapshot := mapAZTStatus(resp.Data[0], 0)
	if len(resp.Data) > 1 {
		snapshot = mapAZTStatus(resp.Data[0], resp.Data[1])
	}
	return snapshot, nil
}

func (c *AZTMasterClient) SetPrice(ctx context.Context, amountMinor int64) error {
	resp, err := c.exchange(ctx, aztCmdSetPrice, mustEncodeMinorUnits(amountMinor, 4))
	if err != nil {
		return err
	}
	return expectAZTAck(resp, "set price")
}

func (c *AZTMasterClient) SetAmountDose(ctx context.Context, amountMinor int64) error {
	resp, err := c.exchange(ctx, aztCmdSetAmountDose, mustEncodeMinorUnits(amountMinor, 6))
	if err != nil {
		return err
	}
	return expectAZTAck(resp, "set amount dose")
}

func (c *AZTMasterClient) SetLitersDose(ctx context.Context, liters float64) error {
	units := int64(liters*100 + 0.5)
	payload, err := encodeDigits(units, 5)
	if err != nil {
		return err
	}
	resp, err := c.exchange(ctx, aztCmdSetLitersDose, payload)
	if err != nil {
		return err
	}
	return expectAZTAck(resp, "set liters dose")
}

func (c *AZTMasterClient) Authorize(ctx context.Context) error {
	resp, err := c.exchange(ctx, aztCmdAuthorize, nil)
	if err != nil {
		return err
	}
	return expectAZTAck(resp, "authorize")
}

func (c *AZTMasterClient) ReadCurrentVolume(ctx context.Context) (float64, error) {
	resp, err := c.exchange(ctx, aztCmdCurrentVolume, nil)
	if err != nil {
		return 0, err
	}
	if resp.ShortResponse != nil {
		return 0, fmt.Errorf("unexpected short response for current volume")
	}
	if len(resp.Data) != 6 || resp.Data[0] != '0' {
		return 0, fmt.Errorf("unexpected current volume payload")
	}
	value, err := decodeDigits(resp.Data[1:])
	if err != nil {
		return 0, err
	}
	return float64(value) / 100, nil
}

func (c *AZTMasterClient) ReadTotals(ctx context.Context) (AZTDeliveryTotals, error) {
	resp, err := c.exchange(ctx, aztCmdTotals, nil)
	if err != nil {
		return AZTDeliveryTotals{}, err
	}
	if resp.ShortResponse != nil {
		return AZTDeliveryTotals{}, fmt.Errorf("unexpected short response for totals")
	}
	if len(resp.Data) < 18 {
		return AZTDeliveryTotals{}, fmt.Errorf("unexpected totals payload length")
	}

	litersDigits := resp.Data[:6]
	amountDigits := resp.Data[6:14]
	priceDigits := resp.Data[14:]

	liters, err := decodeDigits(litersDigits)
	if err != nil {
		return AZTDeliveryTotals{}, err
	}
	amountMinor, err := decodeDigits(amountDigits)
	if err != nil {
		return AZTDeliveryTotals{}, err
	}
	priceMinor, err := decodeDigits(priceDigits)
	if err != nil {
		return AZTDeliveryTotals{}, err
	}

	return AZTDeliveryTotals{
		DispensedLiters: float64(liters) / 100,
		AmountRub:       amountMinor / 100,
		PriceRub:        priceMinor / 100,
	}, nil
}

func (c *AZTMasterClient) ConfirmTotals(ctx context.Context) error {
	resp, err := c.exchange(ctx, aztCmdConfirmTotals, nil)
	if err != nil {
		return err
	}
	return expectAZTAck(resp, "confirm totals")
}

func (c *AZTMasterClient) exchange(ctx context.Context, command byte, data []byte) (AZTResponse, error) {
	frame, err := EncodeAZTRequest(AZTRequest{
		StartByte: c.startByte,
		Address:   c.address,
		Command:   command,
		Data:      data,
	})
	if err != nil {
		return AZTResponse{}, err
	}

	raw, err := c.transport.Exchange(ctx, frame)
	if err != nil {
		return AZTResponse{}, err
	}
	return DecodeAZTResponse(raw)
}

func mapAZTStatus(status byte, reason byte) AZTStatusSnapshot {
	snapshot := AZTStatusSnapshot{
		StatusCode: status,
		ReasonCode: reason,
	}

	switch status {
	case '0':
		snapshot.ProviderStatus = "idle_nozzle_down"
	case '1':
		snapshot.ProviderStatus = "idle_nozzle_up"
	case '2':
		snapshot.ProviderStatus = "authorized"
	case '3':
		snapshot.ProviderStatus = "dispensing"
	case '4':
		snapshot.ProviderStatus = "completed"
		snapshot.Completed = true
		snapshot.Partial = reason == '1'
	case '8':
		snapshot.ProviderStatus = "dose_from_local_panel"
	default:
		snapshot.ProviderStatus = "unknown"
	}

	return snapshot
}

func mustEncodeMinorUnits(amountMinor int64, width int) []byte {
	digits, err := encodeDigits(amountMinor, width)
	if err != nil {
		return nil
	}
	return digits
}

func expectAZTAck(resp AZTResponse, operation string) error {
	if resp.ShortResponse == nil {
		return fmt.Errorf("%s: expected short response", operation)
	}
	switch *resp.ShortResponse {
	case AZTShortResponseACK:
		return nil
	case AZTShortResponseCAN:
		return fmt.Errorf("%s: command rejected in current state", operation)
	case AZTShortResponseNAK:
		return fmt.Errorf("%s: command not supported", operation)
	default:
		return fmt.Errorf("%s: unexpected short response", operation)
	}
}
