package utils

import (
	"errors"
	"fmt"
	"github.com/goastian/astiango-hub/trace"
	"io"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
)

var logger = NewLogger("Utils")

// Logger represents a logger with a specific service prefix.
type Logger struct {
	prefix string
}

type ServiceLoggerOption func(logger *Logger)

func WithHandler(handlerType string, output *os.File) ServiceLoggerOption {
	return func(logger *Logger) {
		SetHandler(handlerType, output)
	}
}

// NewLogger creates a new logger with the given service name as a prefix.
func NewLogger(prefix string, opts ...ServiceLoggerOption) *Logger {
	logger := &Logger{
		prefix: prefix,
	}

	if len(opts) == 0 {
		SetConsoleHandler()
	} else {
		for _, opt := range opts {
			opt(logger)
		}
	}

	return logger
}

// Debug logs a debug message.
func (l *Logger) Debug(message string) {
	log.Debug(l.getFormat(message))
}

// Info logs an informational message.
func (l *Logger) Info(message string) {
	log.Info(l.getFormat(message))
}

// Warn logs a warning message.
func (l *Logger) Warn(message string) {
	log.Warn(l.getFormat(message))
}

// Error logs an error message.
func (l *Logger) Error(message string) {
	log.Error(l.getFormat(message))
	if IsDev() {
		trace.PrintError(errors.New(message))
	}
}

// Fatal logs a fatal message.
func (l *Logger) Fatal(message string) {
	log.Fatal(l.getFormat(message))
	if IsDev() {
		trace.PrintError(errors.New(message))
	}
}

// Debugf logs a debug message with formatted content.
func (l *Logger) Debugf(format string, args ...interface{}) {
	log.Debugf(l.getFormat(format), args...)
}

// Infof logs an informational message with formatted content.
func (l *Logger) Infof(format string, args ...interface{}) {
	log.Infof(l.getFormat(format), args...)
}

// Warnf logs a warning message with formatted content.
func (l *Logger) Warnf(format string, args ...interface{}) {
	log.Warnf(l.getFormat(format), args...)
}

// Errorf logs an error message with formatted content.
func (l *Logger) Errorf(format string, args ...interface{}) {
	log.Errorf(l.getFormat(format), args...)
	if IsDev() {
		trace.PrintError(errors.New(fmt.Sprintf(format, args...)))
	}
}

// Fatalf logs an error message with formatted content.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	log.Fatalf(l.getFormat(format), args...)
	if IsDev() {
		trace.PrintError(errors.New(fmt.Sprintf(format, args...)))
	}
}

func (l *Logger) getFormat(format string) string {
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] [%s] %s", timestamp, l.prefix, format)
}

type LogHandler struct {
	mu sync.Mutex
}

func handleLog(w io.Writer, e *log.Entry) error {
	color := text.Colors[e.Level]
	level := text.Strings[e.Level]
	names := e.Fields.Names()

	fmt.Fprintf(w, "\033[%dm%6s\033[0m %-25s", color, level, e.Message)

	for _, name := range names {
		fmt.Fprintf(w, " \033[%dm%s\033[0m=%v", color, name, e.Fields.Get(name))
	}

	fmt.Fprintln(w)

	return nil
}

// MultiHandler is a handler that routes logs to stdout/stderr based on level
type MultiHandler struct {
	mu sync.Mutex
}

// NewMultiHandler creates a handler that routes logs to stdout/stderr based on level
func NewMultiHandler() *MultiHandler {
	return &MultiHandler{
		mu: sync.Mutex{},
	}
}

// HandleLog implements log.Handler interface
func (h *MultiHandler) HandleLog(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Route to stderr for warn, error, and fatal
	if e.Level <= log.WarnLevel {
		return handleLog(os.Stdout, e)
	}
	// Route to stdout for debug and info
	return handleLog(os.Stderr, e)
}

type ConsoleHandler struct {
	mu sync.Mutex
}

func NewConsoleHandler() *ConsoleHandler {
	return &ConsoleHandler{
		mu: sync.Mutex{},
	}
}

func (h *ConsoleHandler) HandleLog(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return handleLog(os.Stdout, e)
}

// SetHandler to include the new option
func SetHandler(handlerType string, output *os.File) {
	switch handlerType {
	case "json":
		log.SetHandler(json.New(output))
	case "text":
		log.SetHandler(text.New(output))
	case "split":
		SetMultiHandler()
	default:
		SetConsoleHandler()
	}
}

func SetMultiHandler() {
	log.SetHandler(NewMultiHandler())
}

func SetConsoleHandler() {
	log.SetHandler(NewConsoleHandler())
}
