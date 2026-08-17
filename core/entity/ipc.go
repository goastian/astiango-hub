package entity

// IPCMessage defines the structure for messages exchanged between parent and child processes
type IPCMessage struct {
	Type    string      `json:"type"`    // message type identifier
	Payload interface{} `json:"payload"` // message content
	IPC     bool        `json:"ipc"`     // Add this field to explicitly mark IPC messages
}
