package interfaces

// Logger interface for reporting informational and warning messages.
type Logger interface {
	// Debug logs a debugging message.
	Debug(message string)

	// Info logs an informational message.
	Info(message string)

	// Warn logs a warning message.
	Warn(message string)

	// Error logs an error message.
	Error(message string)

	// Fatal logs a fatal message.
	Fatal(message string)

	// Debugf logs a formatted debugging message.
	Debugf(format string, args ...interface{})

	// Infof logs a formatted informational message.
	Infof(format string, args ...interface{})

	// Warnf logs a formatted warning message.
	Warnf(format string, args ...interface{})

	// Errorf logs a formatted error message.
	Errorf(format string, args ...interface{})

	// Fatalf logs a formatted fatal message.
	Fatalf(format string, args ...interface{})
}
