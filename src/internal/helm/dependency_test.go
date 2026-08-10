package helm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDependencyArgs(t *testing.T) {
	opts := DependencyOptions{ChartPath: "/tmp/chart", SkipRefresh: true}

	assert.Equal(t,
		[]string{"dependency", "build", "--skip-refresh", "/tmp/chart"},
		dependencyArgs("build", opts),
	)
	assert.Equal(t,
		[]string{"dependency", "update", "--skip-refresh", "/tmp/chart"},
		dependencyArgs("update", opts),
	)
}

func TestIsDependencyLockMismatch(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "helm lock mismatch",
			stderr: "Error: the lock file (Chart.lock) is out of sync with the dependencies file (Chart.yaml). Please update the dependencies",
			want:   true,
		},
		{
			name:   "unrelated helm error",
			stderr: "Error: failed to download chart",
		},
		{
			name:   "lock error without update instruction",
			stderr: "Error: could not read Chart.lock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isDependencyLockMismatch(tt.stderr))
		})
	}
}

func TestBuildDependenciesFallsBackToUpdateForLockMismatch(t *testing.T) {
	var calls [][]string
	runner := func(_ context.Context, args []string) (string, error) {
		calls = append(calls, args)
		if len(calls) == 1 {
			return "Chart.lock is out of sync; update the dependencies", errors.New("exit status 1")
		}
		return "", nil
	}

	err := buildDependencies(context.Background(), DependencyOptions{ChartPath: "/tmp/chart"}, runner)

	assert.NoError(t, err)
	assert.Equal(t, [][]string{
		{"dependency", "build", "/tmp/chart"},
		{"dependency", "update", "/tmp/chart"},
	}, calls)
}

func TestBuildDependenciesDoesNotUpdateForUnrelatedFailure(t *testing.T) {
	var calls [][]string
	runner := func(_ context.Context, args []string) (string, error) {
		calls = append(calls, args)
		return "failed to download chart", errors.New("exit status 1")
	}

	err := buildDependencies(context.Background(), DependencyOptions{ChartPath: "/tmp/chart"}, runner)

	assert.Error(t, err)
	assert.Equal(t, [][]string{{"dependency", "build", "/tmp/chart"}}, calls)
}
