package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/repository"
)

var clusterRoleHook = `
{
  "apiVersion": "rbac.authorization.k8s.io/v1",
  "kind": "ClusterRole",
  "metadata": {
    "name": "cluster-role-hook",
    "annotations": {
      "argocd.argoproj.io/hook": "PostSync"
	}
  }
}`

var testPod = `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "foo"
  }
}`

func TestSyncHookProjectPermissions(t *testing.T) {
	syncCtx := newTestSyncCtx(&v1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []v1.APIResource{
			{Name: "pod", Namespaced: true, Kind: "Pod", Group: ""},
		},
	}, &v1.APIResourceList{
		GroupVersion: "rbac.authorization.k8s.io/v1",
		APIResources: []v1.APIResource{
			{Name: "clusterroles", Namespaced: false, Kind: "ClusterRole", Group: "rbac.authorization.k8s.io"},
		},
	})

	syncCtx.kubectl = mockKubectlCmd{}
	syncCtx.manifestInfo = &repository.ManifestResponse{
		Manifests: []string{clusterRoleHook},
	}
	syncCtx.resources = []v1alpha1.ResourceState{{
		TargetState: testPod,
	}}
	syncCtx.proj.Spec.ClusterResourceWhitelist = []v1.GroupKind{}

	syncCtx.syncOp.SyncStrategy = nil
	syncCtx.sync()
	assert.Equal(t, v1alpha1.OperationFailed, syncCtx.opState.Phase)
	assert.Len(t, syncCtx.syncRes.Resources, 0)
	assert.Contains(t, syncCtx.opState.Message, "not permitted in project")

	// Now add the resource to the whitelist and try again. Resource should be created
	syncCtx.proj.Spec.ClusterResourceWhitelist = []v1.GroupKind{
		{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole"},
	}
	syncCtx.syncOp.SyncStrategy = nil
	syncCtx.sync()
	assert.Len(t, syncCtx.syncRes.Resources, 1)
	assert.Equal(t, v1alpha1.ResourceDetailsSynced, syncCtx.syncRes.Resources[0].Status)
}
