package chunkedbody

// Settings is a set of limitations for parser
type Settings struct {
	MaxChunkSize int64
}

// DefaultSettings returns prepared Settings instance, filled with default values.
// Default values may not always be the most optimal ones
func DefaultSettings() Settings {
	return Settings{
		MaxChunkSize: 1 * 1024 * 1024, // 1mb
	}
}
