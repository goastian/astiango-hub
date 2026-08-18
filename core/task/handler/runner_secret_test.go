package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunnerRedactsInjectedSecretValues(t *testing.T) {
	runner := &Runner{secretValues: []string{"super-secret", "another-secret"}}
	require.Equal(t, "token=[REDACTED] [REDACTED]", runner.redactSecrets("token=super-secret another-secret"))
}
