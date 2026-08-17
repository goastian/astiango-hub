package utils

import (
	"github.com/goastian/astiango-hub/core/constants"
)

func IsCancellable(status string) bool {
	switch status {
	case constants.TaskStatusPending,
		constants.TaskStatusAssigned,
		constants.TaskStatusRunning:
		return true
	default:
		return false
	}
}
