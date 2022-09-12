package types

// Score is a scorecard score
type Score struct {
	Score  float64        `json:"score"`
	Checks map[string]int `json:"checks,omitempty"`
}
