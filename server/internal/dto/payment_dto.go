package dto

import "AUTO-GAS-STATION/server/internal/model"

type PaymentStartResponse struct {
	Transaction *model.Transaction `json:"transaction"`
}

type PaymentStatusResponse struct {
	Transaction *model.Transaction `json:"transaction"`
}
