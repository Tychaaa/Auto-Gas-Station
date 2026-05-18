package model

import "time"

type Dispenser struct {
	ID        int
	FuelType  string
	Label     string
	UpdatedAt time.Time
}
