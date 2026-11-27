/*
Copyright 2026 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8sresourcewatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubevela/kube-trigger/pkg/eventhandler"
)

func TestTriggersOnDifferentClustersKeepTheirOwnClusters(t *testing.T) {
	w := (&K8sResourceWatcher{}).New().(*K8sResourceWatcher)
	for _, properties := range []string{
		`{"apiVersion":"apps/v1","kind":"Deployment","clusters":["cluster-a"]}`,
		`{"apiVersion":"apps/v1","kind":"Deployment","clusters":["cluster-b"]}`,
	} {
		require.NoError(t, w.Init(&runtime.RawExtension{Raw: []byte(properties)}, eventhandler.New()))
	}

	// Each trigger's handler must be called for its own clusters only.
	watched := map[string]int{}
	for key, config := range w.configs {
		require.Len(t, config.Clusters, 1)
		watched[config.Clusters[0]] = len(w.eventHandlers[key])
	}
	assert.Equal(t, map[string]int{"cluster-a": 1, "cluster-b": 1}, watched)
}
