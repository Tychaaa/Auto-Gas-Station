package dto

type SetMaintenanceRequest struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}
