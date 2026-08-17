package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goastian/astiango-hub/grpc"
)

// writeLogLines marshals log lines to JSON and sends them to the task service
// Uses connection-safe approach for robust task execution
func (r *Runner) writeLogLines(lines []string) {
	// Check if context is cancelled or connection is closed
	select {
	case <-r.ctx.Done():
		return
	default:
	}

	// Check circuit breaker for log connections
	if !r.isLogCircuitClosed() {
		// Circuit is open, don't attempt to send logs to prevent flooding
		return
	}

	// Use connection with mutex for thread safety
	r.connMutex.RLock()
	conn := r.conn
	r.connMutex.RUnlock()

	// Check if connection is available
	if conn == nil {
		r.Debugf("no connection available for sending log lines")
		r.recordLogFailure()
		return
	}

	linesBytes, err := json.Marshal(lines)
	if err != nil {
		r.Errorf("error marshaling log lines: %v", err)
		return
	}

	msg := &grpc.TaskServiceConnectRequest{
		Code:   grpc.TaskServiceConnectCode_INSERT_LOGS,
		TaskId: r.tid.Hex(),
		Data:   linesBytes,
	}

	if err := conn.Send(msg); err != nil {
		// Don't log errors if context is cancelled (expected during shutdown)
		select {
		case <-r.ctx.Done():
			return
		default:
			// Only log the error if circuit breaker allows it to prevent flooding
			if r.isLogCircuitClosed() {
				r.Errorf("error sending log lines: %v", err)
			}
			// Record failure and open circuit breaker if needed
			r.recordLogFailure()
			// Mark connection as unhealthy for reconnection
			r.lastConnCheck = time.Time{}
		}
		return
	}

	// Success - reset circuit breaker
	r.recordLogSuccess()
}

// logInternally sends internal runner logs to the same logging system as the task
func (r *Runner) logInternally(level string, message string) {
	// Format the internal log with a prefix
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05")

	// Pad level
	level = fmt.Sprintf("%-5s", level)

	// Format the log message
	internalLog := fmt.Sprintf("%s [%s] [%s] %s", level, timestamp, "AstianGO Hub", message)

	// Send to the same log system as task logs
	// Only send if context is not cancelled and connection is available
	// AND circuit breaker allows it (prevents cascading log failures)
	if r.conn != nil && r.isLogCircuitClosed() {
		select {
		case <-r.ctx.Done():
			// Context cancelled, don't send logs
		default:
			go r.writeLogLines([]string{internalLog})
		}
	}

	// Also log through the standard logger
	switch level {
	case "ERROR":
		r.Logger.Error(message)
	case "WARN":
		r.Logger.Warn(message)
	case "INFO":
		r.Logger.Info(message)
	case "DEBUG":
		r.Logger.Debug(message)
	}
}

func (r *Runner) Error(message string) {
	msg := fmt.Sprintf(message)
	r.logInternally("ERROR", msg)
}

func (r *Runner) Warn(message string) {
	msg := fmt.Sprintf(message)
	r.logInternally("WARN", msg)
}

func (r *Runner) Info(message string) {
	msg := fmt.Sprintf(message)
	r.logInternally("INFO", msg)
}

func (r *Runner) Debug(message string) {
	msg := fmt.Sprintf(message)
	r.logInternally("DEBUG", msg)
}

func (r *Runner) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	r.logInternally("ERROR", msg)
}

func (r *Runner) Warnf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	r.logInternally("WARN", msg)
}

func (r *Runner) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	r.logInternally("INFO", msg)
}

func (r *Runner) Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	r.logInternally("DEBUG", msg)
}

// Circuit breaker methods for log connection management

// isLogCircuitClosed checks if the circuit breaker allows log sending
func (r *Runner) isLogCircuitClosed() bool {
	r.logConnMutex.RLock()
	defer r.logConnMutex.RUnlock()

	// If circuit was opened due to failures, check if enough time has passed to retry
	if !r.logConnHealthy {
		if time.Since(r.logCircuitOpenTime) > r.logCircuitOpenDuration {
			// Time to retry - close the circuit
			r.logConnMutex.RUnlock()
			r.logConnMutex.Lock()
			r.logConnHealthy = true
			r.logFailureCount = 0
			r.logConnMutex.Unlock()
			r.logConnMutex.RLock()
			return true
		}
		return false
	}

	return true
}

// recordLogFailure records a log sending failure and opens circuit if threshold reached
func (r *Runner) recordLogFailure() {
	r.logConnMutex.Lock()
	defer r.logConnMutex.Unlock()

	r.logFailureCount++
	r.lastLogSendFailure = time.Now()

	// Open circuit breaker after 3 consecutive failures to prevent log flooding
	if r.logFailureCount >= 3 && r.logConnHealthy {
		r.logConnHealthy = false
		r.logCircuitOpenTime = time.Now()
		// Log this through standard logger only (not through writeLogLines to avoid recursion)
		r.Logger.Warn(fmt.Sprintf("log circuit breaker opened after %d failures, suppressing log sends for %v",
			r.logFailureCount, r.logCircuitOpenDuration))
	}
}

// recordLogSuccess records a successful log send and resets the circuit breaker
func (r *Runner) recordLogSuccess() {
	r.logConnMutex.Lock()
	defer r.logConnMutex.Unlock()

	if !r.logConnHealthy || r.logFailureCount > 0 {
		// Circuit was open or had failures, now closing it
		if !r.logConnHealthy {
			r.Logger.Info("log circuit breaker closed - connection restored")
		}
		r.logConnHealthy = true
		r.logFailureCount = 0
	}
}
