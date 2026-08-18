package utils

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type TaskSandboxConfig struct {
	Image       string
	User        string
	Network     string
	CPUs        string
	Memory      string
	PIDs        int
	Disk        string
	Timeout     time.Duration
	TmpfsSize   string
	AppArmor    string
	Seccomp     string
	WorkspaceHost string
}

func GetTaskSandboxConfig() TaskSandboxConfig {
	config := TaskSandboxConfig{
		Image:     viper.GetString("task.sandbox.image"),
		User:      viper.GetString("task.sandbox.user"),
		Network:   viper.GetString("task.sandbox.network"),
		CPUs:      viper.GetString("task.sandbox.cpus"),
		Memory:    viper.GetString("task.sandbox.memory"),
		PIDs:      viper.GetInt("task.sandbox.pids"),
		Disk:      viper.GetString("task.sandbox.disk"),
		Timeout:   viper.GetDuration("task.sandbox.timeout"),
		TmpfsSize: viper.GetString("task.sandbox.tmpfs_size"),
		AppArmor:  viper.GetString("task.sandbox.apparmor"),
		Seccomp:   viper.GetString("task.sandbox.seccomp"),
		WorkspaceHost: viper.GetString("task.sandbox.workspace_host"),
	}
	if config.Image == "" { config.Image = "goastian/astiango-hub-base:sec-009-011" }
	if config.User == "" { config.User = "1000:1000" }
	if config.Network == "" { config.Network = "none" }
	if config.CPUs == "" { config.CPUs = "1" }
	if config.Memory == "" { config.Memory = "512m" }
	if config.PIDs == 0 { config.PIDs = 128 }
	if config.Disk == "" { config.Disk = "1g" }
	if config.Timeout == 0 { config.Timeout = 30 * time.Minute }
	if config.TmpfsSize == "" { config.TmpfsSize = "64m" }
	return config
}

// BuildSandboxedTaskCommand creates a fail-closed Docker job command. The
// host process only invokes Docker; crawler code runs in an ephemeral, named
// container with no inherited host environment or network by default.
func BuildSandboxedTaskCommand(ctx context.Context, taskID, workspace string, environment []string, command string) (*exec.Cmd, string, error) {
	config := GetTaskSandboxConfig()
	if !filepath.IsAbs(workspace) || taskID == "" || command == "" {
		return nil, "", fmt.Errorf("invalid sandbox task configuration")
	}
	name := "astiango-task-" + taskID
	args := []string{"run", "--rm", "--init", "--name", name,
		"--read-only", "--user", config.User, "--network", config.Network,
		"--cap-drop", "ALL", "--security-opt", "no-new-privileges",
		"--pids-limit", strconv.Itoa(config.PIDs), "--memory", config.Memory,
		"--memory-swap", config.Memory, "--cpus", config.CPUs,
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=" + config.TmpfsSize,
		"--workdir", "/workspace",
	"--mount", "type=bind,src=" + sandboxWorkspaceSource(config.WorkspaceHost, workspace) + ",dst=/workspace"}
	if config.Disk != "" { args = append(args, "--storage-opt", "size="+config.Disk) }
	if config.AppArmor != "" { args = append(args, "--security-opt", "apparmor="+config.AppArmor) }
	if config.Seccomp != "" { args = append(args, "--security-opt", "seccomp="+config.Seccomp) }
	for _, value := range environment { args = append(args, "--env", value) }
	args = append(args, config.Image, "/bin/sh", "-lc", command)
	timed, _ := context.WithTimeout(ctx, config.Timeout)
	return exec.CommandContext(timed, "docker", args...), name, nil
}

func sandboxWorkspaceSource(hostRoot, workspace string) string {
	if hostRoot == "" {
		return workspace
	}
	base := GetWorkspace()
	relative, err := filepath.Rel(base, workspace)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return workspace
	}
	return filepath.Join(hostRoot, relative)
}

func RemoveSandbox(name string) {
	if name == "" { return }
	_ = exec.Command("docker", "rm", "-f", name).Run()
}

func StopSandbox(name string, force bool) {
	if name == "" { return }
	if force {
		RemoveSandbox(name)
		return
	}
	_ = exec.Command("docker", "stop", "--time", "15", name).Run()
}
