package types

import "github.com/ossf/scorecard-webapp/app/generated/models"

// Result is the scorecard result for a repository, with any associated packages
type Result struct {
	Repository Repository              `json:"repository,omitempty"`
	Packages   []Package               `json:"packages,omitempty"`
	Result     *models.ScorecardResult `json:"result,omitempty"`
}
