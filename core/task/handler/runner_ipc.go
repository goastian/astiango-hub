package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/grpc"
)

// handleIPC processes incoming IPC messages from the child process
// Messages are converted to JSON and written to the child process's stdin
func (r *Runner) handleIPC() {
	r.wg.Add(1)
	defer r.wg.Done()

	for msg := range r.ipcChan {
		// Convert message to JSON
		jsonData, err := json.Marshal(msg)
		if err != nil {
			r.Errorf("error marshaling IPC message: %v", err)
			continue
		}

		// Write to child process's stdin
		_, err = fmt.Fprintln(r.stdinPipe, string(jsonData))
		if err != nil {
			r.Errorf("error writing to child process: %v", err)
		}
	}
}

// SetIPCHandler sets the handler for incoming IPC messages
func (r *Runner) SetIPCHandler(handler func(entity.IPCMessage)) {
	r.ipcHandler = handler
}

// startIPCReader continuously reads IPC messages from the child process's stdout
// Messages are parsed and either handled by the IPC handler or written to logs
func (r *Runner) startIPCReader() {
	r.wg.Add(2) // Add 2 to wait group for both stdout and stderr readers

	// Start stdout reader
	go func() {
		defer r.wg.Done()
		r.readOutput(r.readerStdout, true) // true for stdout
	}()

	// Start stderr reader
	go func() {
		defer r.wg.Done()
		r.readOutput(r.readerStderr, false) // false for stderr
	}()
}

// handleIPCInsertDataMessage converts the IPC message payload to JSON and sends it to the master node
func (r *Runner) handleIPCInsertDataMessage(ipcMsg entity.IPCMessage) {
	if ipcMsg.Payload == nil {
		r.Errorf("empty payload in IPC message")
		return
	}

	// Convert payload to data to be inserted
	var records []map[string]interface{}

	switch payload := ipcMsg.Payload.(type) {
	case []interface{}: // Handle array of objects
		records = make([]map[string]interface{}, 0, len(payload))
		for i, item := range payload {
			if itemMap, ok := item.(map[string]interface{}); ok {
				records = append(records, itemMap)
			} else {
				r.Errorf("invalid record at index %d: %v", i, item)
				continue
			}
		}
	case []map[string]interface{}: // Handle direct array of maps
		records = payload
	case map[string]interface{}: // Handle single object
		records = []map[string]interface{}{payload}
	case interface{}: // Handle generic interface
		if itemMap, ok := payload.(map[string]interface{}); ok {
			records = []map[string]interface{}{itemMap}
		} else {
			r.Errorf("invalid payload type: %T", payload)
			return
		}
	default:
		r.Errorf("unsupported payload type: %T, value: %v", payload, ipcMsg.Payload)
		return
	}

	// Validate records
	if len(records) == 0 {
		r.Warnf("no valid records to insert")
		return
	}

	// Marshal data with error handling
	data, err := json.Marshal(records)
	if err != nil {
		r.Errorf("error marshaling records: %v", err)
		return
	}

	// Check if context is cancelled or connection is closed
	select {
	case <-r.ctx.Done():
		return
	default:
	}

	// Use connection with mutex for thread safety
	r.connMutex.RLock()
	conn := r.conn
	r.connMutex.RUnlock()

	// Validate connection
	if conn == nil {
		r.Errorf("gRPC connection not initialized")
		return
	}

	// Send IPC message to master with context and timeout - use runner's context
	ctx, cancel := context.WithTimeout(r.ctx, r.ipcTimeout)
	defer cancel()

	// Create gRPC message
	grpcMsg := &grpc.TaskServiceConnectRequest{
		Code:   grpc.TaskServiceConnectCode_INSERT_DATA,
		TaskId: r.tid.Hex(),
		Data:   data,
	}

	// Use context for sending
	select {
	case <-ctx.Done():
		r.Errorf("timeout sending IPC message")
		return
	case <-r.ctx.Done():
		return
	default:
		if err := conn.Send(grpcMsg); err != nil {
			// Don't log errors if context is cancelled (expected during shutdown)
			select {
			case <-r.ctx.Done():
				return
			default:
				r.Errorf("error sending IPC message: %v", err)
				// Mark connection as unhealthy for reconnection
				r.lastConnCheck = time.Time{}
			}
			return
		}

		// Update last successful connection time to help health check avoid unnecessary pings
		r.lastConnCheck = time.Now()
	}
}

func (r *Runner) readOutput(reader *bufio.Reader, isStdout bool) {
	scanner := bufio.NewScanner(reader)
	for {
		select {
		case <-r.ctx.Done():
			// Context cancelled, stop reading
			return
		default:
			// Scan the next line
			if !scanner.Scan() {
				return
			}

			// Get the line
			line := scanner.Text()

			// Trim the line
			line = strings.TrimRight(line, "\n\r")

			// For stdout, try to parse as IPC message first
			if isStdout {
				var ipcMsg entity.IPCMessage
				if err := json.Unmarshal([]byte(line), &ipcMsg); err == nil && ipcMsg.IPC {
					if r.ipcHandler != nil {
						r.ipcHandler(ipcMsg)
					} else {
						// Default handler (insert data)
						if ipcMsg.Type == "" || ipcMsg.Type == constants.IPCMessageData {
							r.handleIPCInsertDataMessage(ipcMsg)
						} else {
							r.Warnf("no IPC handler set for message: %v", ipcMsg)
						}
					}
					continue
				}
			}

			// If not an IPC message or from stderr, treat as log
			r.writeLogLines([]string{r.redactSecrets(line)})
		}
	}
}
