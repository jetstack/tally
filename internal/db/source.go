package db

import "context"

// Source populates the database with data
type Source interface {
	// String returns a string that identifies the source
	String() string

	// Update the database with items from the source
	Update(context.Context) error
}
