package dto

type SetMaintenanceRequest struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type SetScreenRequest struct {
	Screen string `json:"screen" binding:"required,max=64"`
}
