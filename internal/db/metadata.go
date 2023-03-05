package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Metadata is metadata about the database
type Metadata struct {
	SHA256 string `json:"sha256"`
}

// Equals comapres metadata
func (m Metadata) Equals(metadata Metadata) bool {
	return m.SHA256 == metadata.SHA256
}

// Metadata returns metadata about the current database
func (m *manager) Metadata() (*Metadata, error) {
	f, err := os.Open(filepath.Join(m.dbDir, "metadata.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("opening metadata file: %w", err)
	}
	defer f.Close()

	metadata := &Metadata{}
	if err := json.NewDecoder(f).Decode(metadata); err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	return metadata, nil
}
