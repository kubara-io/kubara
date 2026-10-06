package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanManifest(t *testing.T) {
	manifest := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: main-config
---
apiVersion: batch/v1
kind: Job
metadata:
  name: pre-job-high-weight
  annotations:
    helm.sh/hook: pre-install
    helm.sh/hook-weight: "10"
---
apiVersion: batch/v1
kind: Job
metadata:
  name: pre-job-low-weight
  annotations:
    helm.sh/hook: pre-upgrade
    helm.sh/hook-weight: "-5"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: main-deployment
---
apiVersion: batch/v1
kind: Job
metadata:
  name: post-job
  annotations:
    helm.sh/hook: post-install, post-upgrade
---
apiVersion: batch/v1
kind: Job
metadata:
  name: delete-job
  annotations:
    helm.sh/hook: pre-delete
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  annotations:
    helm.sh/hook: test
`)

	plan, err := PlanManifest(manifest)
	require.NoError(t, err)

	// PreInstall checks: pre-upgrade and pre-install both present, sorted by weight (-5 before 10)
	require.Len(t, plan.PreInstall, 2)
	assert.Equal(t, "pre-job-low-weight", plan.PreInstall[0].GetName())
	assert.Equal(t, "pre-job-high-weight", plan.PreInstall[1].GetName())

	// Main checks: un-annotated resources only
	require.Len(t, plan.Main, 2)
	assert.Equal(t, "main-config", plan.Main[0].GetName())
	assert.Equal(t, "main-deployment", plan.Main[1].GetName())

	// PostInstall checks
	require.Len(t, plan.PostInstall, 1)
	assert.Equal(t, "post-job", plan.PostInstall[0].GetName())

	// PreDelete checks
	require.Len(t, plan.PreDelete, 1)
	assert.Equal(t, "delete-job", plan.PreDelete[0].GetName())

	// PostDelete should be empty
	assert.Empty(t, plan.PostDelete)
}

func TestPlanManifest_InvalidWeight(t *testing.T) {
	manifest := []byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: invalid-job
  annotations:
    helm.sh/hook: pre-install
    helm.sh/hook-weight: "not-a-number"
`)

	plan, err := PlanManifest(manifest)
	require.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "sort pre-install hooks")
}
