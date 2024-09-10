package activity

import (
	"time"
)

type Split struct {
	Distance  float64
	SplitTime float64
	Elevation float64
	HeartRate float64
}

type Activity struct {
	Name          string
	Created       time.Time
	GPX           []byte
	Distance      float64
	Time          float64
	CompletedDate *time.Time
	Elevation     float64
	Uphill        float64
	Downhill      float64
	AveragePace   float64
	Splits        []Split
}
