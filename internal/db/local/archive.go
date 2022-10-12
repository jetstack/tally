package local

// Archive is an archive of the local database
type Archive interface {
	// DB returns the database data from the archive
	DB() ([]byte, error)

	// Metadata returns metadata about the database in the archive
	Metadata() (*Metadata, error)
}
