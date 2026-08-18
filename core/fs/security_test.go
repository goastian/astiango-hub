package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceConfinementAppliesToAllMutations(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	svc := NewFsService(root)

	for _, operation := range []func() error{
		func() error { return svc.Save("../outside", []byte("x")) },
		func() error { return svc.CreateDir("../outside") },
		func() error { return svc.Delete("../outside") },
		func() error { return svc.Rename("../a", "b") },
		func() error { return svc.Copy("../a", "b") },
	} {
		require.Error(t, operation())
	}

	require.NoError(t, os.Symlink(outside, filepath.Join(root, "escape")))
	require.Error(t, svc.Save("escape/file", []byte("x")))
}
