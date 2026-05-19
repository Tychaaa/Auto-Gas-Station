package model

import "time"

type Dispenser struct {
	ID        int
	FuelType  string
	Label     string
	Enabled   bool
	SortOrder int
	UpdatedAt time.Time
}
