package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessIdExists(t *testing.T) {
	t.Run("existing process", func(t *testing.T) {
		currentPid := os.Getpid()
		assert.True(t, ProcessIdExists(currentPid), "should detect current process")
	})

	t.Run("non-existent process", func(t *testing.T) {
		invalidPid := 99999999
		assert.False(t, ProcessIdExists(invalidPid), "should not detect non-existent process")
	})
}
