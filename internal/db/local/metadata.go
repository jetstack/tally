package local

// Metadata is metadata about the database
type Metadata struct {
	SHA256 string `json:"sha256"`
}

// Equals comapres metadata
func (m Metadata) Equals(metadata Metadata) bool {
	return m.SHA256 == metadata.SHA256
}
