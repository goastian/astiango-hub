package handler

import (
	"runtime"
	"syscall"
	"time"

	"github.com/goastian/astiango-hub/core/utils"
	"github.com/shirou/gopsutil/process"
)

// cleanupOrphanedProcesses attempts to clean up any orphaned processes related to this task
func (r *Runner) cleanupOrphanedProcesses() {
	r.Warnf("cleaning up orphaned processes for task %s (PID: %d)", r.tid.Hex(), r.pid)

	if r.pid <= 0 {
		r.Debugf("no PID to clean up")
		return
	}

	// Try to kill the process group if it exists
	if runtime.GOOS != "windows" {
		r.killProcessGroup()
	}

	// Force kill the main process if it still exists
	if utils.ProcessIdExists(r.pid) {
		r.Warnf("forcefully killing remaining process %d", r.pid)
		if r.cmd != nil && r.cmd.Process != nil {
			if err := utils.KillProcess(r.cmd, true); err != nil {
				r.Errorf("failed to force kill process: %v", err)
			}
		}
	}

	// Scan for any remaining child processes and kill them
	r.scanAndKillChildProcesses()
}

// killProcessGroup kills the entire process group on Unix systems
func (r *Runner) killProcessGroup() {
	if r.pid <= 0 {
		return
	}

	r.Debugf("attempting to kill process group for PID %d", r.pid)

	// Kill the process group (negative PID kills the group)
	err := syscall.Kill(-r.pid, syscall.SIGTERM)
	if err != nil {
		r.Debugf("failed to send SIGTERM to process group: %v", err)
		// Try SIGKILL as last resort
		err = syscall.Kill(-r.pid, syscall.SIGKILL)
		if err != nil {
			r.Debugf("failed to send SIGKILL to process group: %v", err)
		}
	} else {
		r.Debugf("successfully sent SIGTERM to process group %d", r.pid)
	}
}

// scanAndKillChildProcesses scans for and kills any remaining child processes
func (r *Runner) scanAndKillChildProcesses() {
	r.Debugf("scanning for orphaned child processes of task %s", r.tid.Hex())

	processes, err := utils.GetProcesses()
	if err != nil {
		r.Errorf("failed to get process list: %v", err)
		return
	}

	taskIdEnv := "ASTIANGO_TASK_ID=" + r.tid.Hex()
	killedCount := 0

	for _, proc := range processes {
		// Check if this process has our task ID in its environment
		if r.isTaskRelatedProcess(proc, taskIdEnv) {
			pid := int(proc.Pid)
			r.Warnf("found orphaned task process PID %d, killing it", pid)

			// Kill the orphaned process
			if err := proc.Kill(); err != nil {
				r.Errorf("failed to kill orphaned process %d: %v", pid, err)
			} else {
				killedCount++
				r.Infof("successfully killed orphaned process %d", pid)
			}
		}
	}

	if killedCount > 0 {
		r.Infof("cleaned up %d orphaned processes for task %s", killedCount, r.tid.Hex())
	} else {
		r.Debugf("no orphaned processes found for task %s", r.tid.Hex())
	}
}

// isTaskRelatedProcess checks if a process is related to this task
func (r *Runner) isTaskRelatedProcess(proc *process.Process, taskIdEnv string) bool {
	// Get process environment variables
	environ, err := proc.Environ()
	if err != nil {
		// If we can't read environment, skip this process
		return false
	}

	// Check if this process has our task ID
	for _, env := range environ {
		if env == taskIdEnv {
			return true
		}
	}

	return false
}

// startZombieMonitor starts a background goroutine to monitor for zombie processes
func (r *Runner) startZombieMonitor() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		// Check for zombies every 5 minutes
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				r.checkForZombieProcesses()
			}
		}
	}()
}

// checkForZombieProcesses periodically checks for and cleans up zombie processes
func (r *Runner) checkForZombieProcesses() {
	r.Debugf("checking for zombie processes related to task %s", r.tid.Hex())

	// Check if our main process still exists and is in the expected state
	if r.pid > 0 && utils.ProcessIdExists(r.pid) {
		// Process exists, check if it's a zombie
		if proc, err := process.NewProcess(int32(r.pid)); err == nil {
			if status, err := proc.Status(); err == nil {
				if status == "Z" || status == "zombie" {
					r.Warnf("detected zombie process %d for task %s", r.pid, r.tid.Hex())
					go r.cleanupOrphanedProcesses()
				}
			}
		}
	}
}

// performPeriodicCleanup runs periodic cleanup for all tasks
func (r *Runner) performPeriodicCleanup() {
	r.wg.Add(1)
	defer r.wg.Done()

	// Cleanup every 10 minutes for all tasks
	r.resourceCleanup = time.NewTicker(10 * time.Minute)
	defer r.resourceCleanup.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.resourceCleanup.C:
			r.runPeriodicCleanup()
		}
	}
}

// runPeriodicCleanup performs memory and resource cleanup
func (r *Runner) runPeriodicCleanup() {
	r.Debugf("performing periodic cleanup for task")

	// Force garbage collection for memory management
	runtime.GC()

	// Log current resource usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	r.Debugf("memory usage - alloc: %d KB, sys: %d KB, num_gc: %d",
		m.Alloc/1024, m.Sys/1024, m.NumGC)

	// Check if IPC channel is getting full
	if r.ipcChan != nil {
		select {
		case <-r.ipcChan:
			r.Debugf("drained stale IPC message during cleanup")
		default:
			// Channel is not full, good
		}
	}
}
