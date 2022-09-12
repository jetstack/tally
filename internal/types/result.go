package types

// Result is the scorecard score for a package+repository combination
type Result struct {
	PackageSystem string `json:"packageSystem"`
	PackageName   string `json:"packageName"`
	Repository    string `json:"repository,omitempty"`
	Score         *Score `json:"score,omitempty"`
}
