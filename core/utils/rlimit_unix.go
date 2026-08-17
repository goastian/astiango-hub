//go:build !windows && !plan9

package utils

import "syscall"

func EnsureFileDescriptorLimit(min uint64) {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("failed to get rlimit: %v", err)
		return
	}

	if rLimit.Cur >= min {
		return
	}

	newLimit := min
	if rLimit.Max < newLimit {
		rLimit.Max = newLimit
	}
	rLimit.Cur = newLimit

	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("failed to raise rlimit to %d: %v", newLimit, err)
		return
	}

	logger.Infof("increased file descriptor limit to %d", newLimit)
}
