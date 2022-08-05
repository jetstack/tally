package tally

// Package represents a supported package
type Package struct {
	Type    PackageType `json:"type"`
	Name    string      `json:"name"`
	Version string      `json:"version"`
}

// PackageType is a supported type of package
type PackageType string

const (
	PackageTypeCargo PackageType = "cargo"
	PackageTypeGo    PackageType = "go"
	PackageTypeMaven PackageType = "maven"
	PackageTypeNPM   PackageType = "npm"
	PackageTypePyPI  PackageType = "pypi"
)

// DepsDevSystem returns the type as an equivalent value in the SYSTEM column
// in the deps.dev BigQuery dataset. If it returns an empty string then the type
// is not supported by deps.dev.
func (t *PackageType) DepsDevSystem() string {
	switch *t {
	case PackageTypeCargo:
		return "CARGO"
	case PackageTypeGo:
		return "GO"
	case PackageTypeMaven:
		return "MAVEN"
	case PackageTypeNPM:
		return "NPM"
	case PackageTypePyPI:
		return "PYPI"
	default:
		return ""
	}
}

func isPackage(a, b Package) bool {
	return a.Type == b.Type && a.Name == b.Name && a.Version == b.Version
}

func containsPackage(pkgs []Package, pkg Package) bool {
	for _, p := range pkgs {
		if isPackage(p, pkg) {
			return true
		}
	}

	return false
}
