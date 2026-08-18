package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePathWithinRootRejectsTraversalAbsoluteAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "escape")))

	_, err := ResolvePathWithinRoot(root, "../outside")
	require.Error(t, err)
	_, err = ResolvePathWithinRoot(root, filepath.Join(outside, "file"))
	require.Error(t, err)
	_, err = ResolvePathWithinRoot(root, "escape/file")
	require.Error(t, err)

	resolved, err := ResolvePathWithinRoot(root, "safe/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "safe", "file.txt"), resolved)
}

func FuzzResolvePathWithinRoot(f *testing.F) {
	f.Add("file.txt")
	f.Add("../escape")
	f.Add("nested/../../escape")
	f.Add("")
	f.Fuzz(func(t *testing.T, requestedPath string) {
		root := t.TempDir()
		resolved, err := ResolvePathWithinRoot(root, requestedPath)
		if err != nil {
			return
		}
		rel, err := filepath.Rel(root, resolved)
		require.NoError(t, err)
		require.NotEqual(t, "..", rel)
	})
}
