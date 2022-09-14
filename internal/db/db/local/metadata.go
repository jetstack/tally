package local

import "time"

// Metadata is information about the database
type Metadata struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}

// Equal compares one Metadata to another
func (m Metadata) Equal(metadata Metadata) bool {
	return m.Hash == metadata.Hash
}
