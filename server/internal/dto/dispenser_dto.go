package dto

type DispenserView struct {
	ID            int    `json:"id"`
	FuelType      string `json:"fuelType"`
	Label         string `json:"label"`
	Enabled       bool   `json:"enabled"`
	TankVolume    int    `json:"tankVolume"`
	TankRemaining int    `json:"tankRemaining"`
	UpdatedAt     string `json:"updatedAt"`
}

type UpdateDispenserRequest struct {
	FuelType string `json:"fuelType"`
	Enabled  bool   `json:"enabled"`
}
