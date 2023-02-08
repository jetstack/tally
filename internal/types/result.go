package types

// Result is the scorecard score for a repository with any packages associated
// with that repository
type Result struct {
	Repository string    `json:"repository,omitempty"`
	Packages   []Package `json:"packages,omitempty"`
	Score      *Score    `json:"score,omitempty"`
}
