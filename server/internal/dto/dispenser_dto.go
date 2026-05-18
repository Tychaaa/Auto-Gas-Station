package dto

type DispenserView struct {
	ID        int    `json:"id"`
	FuelType  string `json:"fuelType"`
	Label     string `json:"label"`
	UpdatedAt string `json:"updatedAt"`
}

type AssignDispenserRequest struct {
	FuelType string `json:"fuelType"`
}
