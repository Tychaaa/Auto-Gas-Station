package dto

type DispenserView struct {
	ID        int    `json:"id"`
	FuelType  string `json:"fuelType"`
	Label     string `json:"label"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updatedAt"`
	// TODO(топливомер): добавить TankVolume и TankRemaining когда будет интеграция с датчиком уровня топлива
}

type UpdateDispenserRequest struct {
	FuelType string `json:"fuelType"`
	Enabled  bool   `json:"enabled"`
}
