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

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
)

func trigger(sourceType, properties string) v1alpha1.TriggerMeta {
	return v1alpha1.TriggerMeta{Source: v1alpha1.Source{
		Type:       sourceType,
		Properties: &runtime.RawExtension{Raw: []byte(properties)},
	}}
}

func noopHandler(v1alpha1.TriggerMeta) (eventhandler.EventHandler, error) {
	return eventhandler.New(), nil
}

func TestEveryCronTriggerGetsASourceToRun(t *testing.T) {
	sources, err := initSources([]v1alpha1.TriggerMeta{
		trigger("cronjob", `{"schedule":"0 1 * * *"}`),
		trigger("cronjob", `{"schedule":"0 2 * * *"}`),
		trigger("cronjob", `{"schedule":"0 3 * * *"}`),
	}, sourceregistry.NewWithBuiltinSources(), noopHandler)
	require.NoError(t, err)
	assert.Len(t, sources, 3)
}

func TestTriggersOfASingletonSourceShareOne(t *testing.T) {
	sources, err := initSources([]v1alpha1.TriggerMeta{
		trigger("resource-watcher", `{"apiVersion":"v1","kind":"ConfigMap"}`),
		trigger("resource-watcher", `{"apiVersion":"apps/v1","kind":"Deployment"}`),
	}, sourceregistry.NewWithBuiltinSources(), noopHandler)
	require.NoError(t, err)
	assert.Len(t, sources, 1)
}
