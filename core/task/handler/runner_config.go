package handler

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/models/client"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// configurePythonPath sets up the Python environment paths, handling both pyenv and default installations
func (r *Runner) configurePythonPath() {
	// Configure global node_modules path
	pyenvRoot := utils.GetPyenvPath()
	pyenvShimsPath := pyenvRoot + "/shims"
	pyenvBinPath := pyenvRoot + "/bin"

	// Configure global pyenv path
	r.cmd.Env = append(r.cmd.Env, "PYENV_ROOT="+pyenvRoot)

	// Update PATH with pyenv paths
	currentPath := r.getEnvFromCmd("PATH")
	newPath := pyenvBinPath + ":" + pyenvShimsPath + ":" + currentPath
	r.setEnvInCmd("PATH", newPath)
}

// configureNodePath sets up the Node.js environment paths, handling both nvm and default installations
func (r *Runner) configureNodePath() {
	// Configure nvm-based Node.js paths
	currentPath := r.getEnvFromCmd("PATH")

	// Configure global node_modules path
	nodePath := utils.GetNodeModulesPath()
	if !strings.Contains(currentPath, nodePath) {
		currentPath = nodePath + ":" + currentPath
		r.setEnvInCmd("PATH", currentPath)
	}
	r.cmd.Env = append(r.cmd.Env, "NODE_PATH="+nodePath)

	// Configure global node_bin path
	nodeBinPath := utils.GetNodeBinPath()
	// Get the updated PATH after the node_modules path was added
	updatedPath := r.getEnvFromCmd("PATH")
	if !strings.Contains(updatedPath, nodeBinPath) {
		newPath := nodeBinPath + ":" + updatedPath
		r.setEnvInCmd("PATH", newPath)
	}
}

func (r *Runner) configureGoPath() {
	// Configure global go path
	goPath := utils.GetGoPath()
	if goPath != "" {
		r.cmd.Env = append(r.cmd.Env, "GOPATH="+goPath)
	}
}

// configureEnv builds a sterile environment and injects only records whose
// tenant, project, or task scope matches the current task.
func (r *Runner) configureEnv() {
	r.cmd = &exec.Cmd{Env: []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/workspace",
		"TMPDIR=/tmp",
	}}

	// Configure Python path
	r.configurePythonPath()

	// Configure Node.js path
	r.configureNodePath()

	// Configure Go path
	r.configureGoPath()

	// Task-specific environment variables
	r.cmd.Env = append(r.cmd.Env, "ASTIANGO_TASK_ID="+r.tid.Hex())

	filters := []bson.M{{"task_id": r.tid}}
	if !r.s.ProjectId.IsZero() {
		filters = append(filters, bson.M{"project_id": r.s.ProjectId})
	}
	tenantID := r.taskTenantID()
	if !tenantID.IsZero() {
		filters = append(filters, bson.M{"tenant_id": tenantID})
	}
	envs, err := client.NewModelService[models.Environment]().GetMany(bson.M{"$or": filters}, nil)
	if err != nil {
		r.Errorf("failed to get environments: %v", err)
	}
	validKey := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	for _, env := range envs {
		if !validKey.MatchString(env.Key) || strings.HasPrefix(env.Key, "ASTIANGO_") {
			r.Warn("skipping invalid scoped environment key")
			continue
		}
		r.cmd.Env = append(r.cmd.Env, env.Key+"="+env.Value)
		r.secretValues = append(r.secretValues, env.Value)
		if _, auditErr := client.NewModelService[models.SecretAccessAudit]().InsertOne(models.SecretAccessAudit{
			TaskId: r.tid, ProjectId: r.s.ProjectId, TenantId: tenantID, Key: env.Key,
		}); auditErr != nil {
			r.Errorf("failed to audit scoped secret access: %v", auditErr)
		}
	}
	r.taskEnv = append([]string(nil), r.cmd.Env...)
}

func (r *Runner) taskTenantID() primitive.ObjectID {
	if r.t == nil || r.t.CreatedBy.IsZero() {
		return primitive.NilObjectID
	}
	u, err := client.NewModelService[models.User]().GetById(r.t.CreatedBy)
	if err != nil || u == nil {
		return primitive.NilObjectID
	}
	return u.TenantId
}

// configureCwd sets the working directory for the task based on the spider's configuration
func (r *Runner) configureCwd() {
	workspacePath := utils.GetWorkspace()
	if r.s.GitId.IsZero() {
		// not git
		r.cwd = filepath.Join(workspacePath, r.s.Id.Hex())
	} else {
		// git
		r.cwd = filepath.Join(workspacePath, r.s.GitId.Hex(), r.s.GitRootPath)
	}
}

// configureCmd builds and configures the command to be executed, including setting up IPC pipes
// and processing command parameters
func (r *Runner) configureCmd() (err error) {
	var cmdStr string

	// command
	if r.t.Cmd == "" {
		cmdStr = r.s.Cmd
	} else {
		cmdStr = r.t.Cmd
	}

	// parameters
	if r.t.Param != "" {
		cmdStr += " " + r.t.Param
	} else if r.s.Param != "" {
		cmdStr += " " + r.s.Param
	}

	// Build a named ephemeral Docker job. Host execution is deliberately not an
	// option for crawler commands.
	r.cmd, r.sandboxName, err = utils.BuildSandboxedTaskCommand(r.ctx, r.tid.Hex(), r.cwd, r.taskEnv, cmdStr)
	if err != nil {
		r.Errorf("error building command: %v", err)
		return err
	}

	// ZOMBIE PREVENTION: Set process group to enable proper cleanup of child processes
	if runtime.GOOS != "windows" {
		// Create new process group on Unix systems to ensure child processes can be killed together
		r.cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group
			Pgid:    0,    // Use process ID as process group ID
		}
	}

	// Configure pipes for IPC and logs
	r.stdinPipe, err = r.cmd.StdinPipe()
	if err != nil {
		r.Errorf("error creating stdin pipe: %v", err)
		return err
	}

	// Add stdout pipe for IPC and logs
	r.stdoutPipe, err = r.cmd.StdoutPipe()
	if err != nil {
		r.Errorf("error creating stdout pipe: %v", err)
		return err
	}

	// Add stderr pipe for error logs
	stderrPipe, err := r.cmd.StderrPipe()
	if err != nil {
		r.Errorf("error creating stderr pipe: %v", err)
		return err
	}

	// Create buffered readers
	r.readerStdout = bufio.NewReader(r.stdoutPipe)
	r.readerStderr = bufio.NewReader(stderrPipe)

	// Initialize IPC channel
	r.ipcChan = make(chan entity.IPCMessage)

	return nil
}

// getEnvFromCmd retrieves an environment variable value from r.cmd.Env
func (r *Runner) getEnvFromCmd(key string) string {
	prefix := key + "="
	for _, env := range r.cmd.Env {
		if after, ok := strings.CutPrefix(env, prefix); ok {
			return after
		}
	}
	return ""
}

// setEnvInCmd sets or updates an environment variable in r.cmd.Env
func (r *Runner) setEnvInCmd(key, value string) {
	envVar := key + "=" + value
	prefix := key + "="

	// Check if the environment variable already exists and update it
	for i, env := range r.cmd.Env {
		if strings.HasPrefix(env, prefix) {
			r.cmd.Env[i] = envVar
			return
		}
	}

	// If not found, append it
	r.cmd.Env = append(r.cmd.Env, envVar)
}
