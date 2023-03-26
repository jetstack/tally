package types

import "github.com/ossf/scorecard-webapp/app/generated/models"

// Result is the scorecard score for a repository with any packages associated
// with that repository
type Result struct {
	Repository      Repository              `json:"repository,omitempty"`
	Packages        []Package               `json:"packages,omitempty"`
	ScorecardResult *models.ScorecardResult `json:"scorecard_result,omitempty"`
}
