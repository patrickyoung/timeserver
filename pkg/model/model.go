package model

import "time"

// TimeResponse represents a time response with various formats
type TimeResponse struct {
	CurrentTime string `json:"current_time"`
	UnixTime    int64  `json:"unix_time"`
	Timezone    string `json:"timezone"`
	Formatted   string `json:"formatted"`
}

// NewTimeResponse creates a new TimeResponse from a time.Time
func NewTimeResponse(t time.Time) *TimeResponse {
	return &TimeResponse{
		CurrentTime: t.Format(time.RFC3339Nano),
		UnixTime:    t.Unix(),
		Timezone:    "UTC",
		Formatted:   t.Format(time.RFC3339),
	}
}
