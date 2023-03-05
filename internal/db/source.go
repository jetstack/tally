package db

import "context"

// Source populates the database with data
type Source interface {
	// String returns the name of the source
	String() string

	// Update the database with items from the source
	Update(context.Context, Writer) error
}
