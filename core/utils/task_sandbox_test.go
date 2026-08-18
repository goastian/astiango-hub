package utils

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestBuildSandboxedTaskCommandUsesRestrictiveDockerFlags(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	cmd, name, err := BuildSandboxedTaskCommand(context.Background(), "abc123", "/tmp/workspace", []string{"TOKEN=top-secret"}, "echo ok")
	require.NoError(t, err)
	require.Equal(t, "astiango-task-abc123", name)
	args := strings.Join(cmd.Args, " ")
	for _, flag := range []string{"--read-only", "--cap-drop ALL", "--security-opt no-new-privileges", "--pids-limit 128", "--memory 512m", "--storage-opt size=1g", "--network none", "--user 1000:1000", "--tmpfs /tmp:rw,noexec,nosuid,size=64m"} {
		require.Contains(t, args, flag)
	}
	require.Contains(t, args, "--env TOKEN=top-secret")
	require.Contains(t, args, "type=bind,src=/tmp/workspace,dst=/workspace")
}

func TestBuildSandboxedTaskCommandRejectsUnsafeInput(t *testing.T) {
	_, _, err := BuildSandboxedTaskCommand(context.Background(), "", "relative", nil, "echo ok")
	require.Error(t, err)
}
