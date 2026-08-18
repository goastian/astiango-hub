package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePathWithinRoot returns a path which is guaranteed to remain under
// root. It rejects absolute and traversal paths and detects existing symlinks
// that would resolve outside root.
func ResolvePathWithinRoot(root, requestedPath string) (string, error) {
	canonicalRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	canonicalRoot, err = filepath.EvalSymlinks(canonicalRoot)
	if err != nil {
		return "", fmt.Errorf("resolve root symlinks: %w", err)
	}
	if filepath.IsAbs(requestedPath) {
		return "", errors.New("absolute paths are not allowed")
	}
	cleanPath := filepath.Clean(requestedPath)
	if cleanPath == "." {
		cleanPath = ""
	}
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}
	candidate := filepath.Join(canonicalRoot, cleanPath)
	if err := ensurePathAncestorWithinRoot(canonicalRoot, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func ensurePathAncestorWithinRoot(root, candidate string) error {
	for probe := candidate; ; probe = filepath.Dir(probe) {
		if _, err := os.Lstat(probe); err == nil {
			resolved, err := filepath.EvalSymlinks(probe)
			if err != nil {
				return fmt.Errorf("resolve path symlinks: %w", err)
			}
			rel, err := filepath.Rel(root, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return errors.New("path escapes its workspace root")
			}
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect path: %w", err)
		}
		if probe == root || filepath.Dir(probe) == probe {
			return errors.New("path has no accessible workspace ancestor")
		}
	}
}
