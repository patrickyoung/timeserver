package logger

// Logger defines the interface for application logging
// This abstraction allows for easier testing and potential logger swapping
type Logger interface {
	// Info logs an informational message with optional key-value pairs
	Info(msg string, args ...any)

	// Error logs an error message with optional key-value pairs
	Error(msg string, args ...any)
}
