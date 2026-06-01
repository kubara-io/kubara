package helm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanDependencies_RemovesLockAndChartsDir(t *testing.T) {
	dir := t.TempDir()

	lockPath := filepath.Join(dir, "Chart.lock")
	chartsDir := filepath.Join(dir, "charts")

	require.NoError(t, os.WriteFile(lockPath, []byte("# stale lock\n"), 0o644))
	require.NoError(t, os.MkdirAll(chartsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(chartsDir, "stale.tgz"), []byte("x"), 0o644))

	require.NoError(t, CleanDependencies(dir))

	_, err := os.Stat(lockPath)
	assert.True(t, os.IsNotExist(err), "Chart.lock should be removed")
	_, err = os.Stat(chartsDir)
	assert.True(t, os.IsNotExist(err), "charts/ should be removed")
}

func TestCleanDependencies_NoArtifactsIsNoop(t *testing.T) {
	dir := t.TempDir()
	// Directory exists, but no Chart.lock and no charts/ — should not error.
	require.NoError(t, CleanDependencies(dir))
}

func TestCleanDependencies_EmptyPathIsNoop(t *testing.T) {
	require.NoError(t, CleanDependencies(""))
}
