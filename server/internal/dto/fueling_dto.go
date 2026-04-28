package dto

type FuelingStartRequest struct {
	PumpID   string `json:"pumpId"`
	NozzleID string `json:"nozzleId"`
	Scenario string `json:"scenario"`
}
