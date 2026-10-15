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

package eventhandler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
)

func object() map[string]interface{} {
	return map[string]interface{}{"metadata": map[string]interface{}{"name": "web"}, "status": map[string]interface{}{"phase": "Pending"}}
}

func TestCELFilterDropsEvents(t *testing.T) {
	testcases := map[string]string{
		"filter is false":              `$(context.data.status.phase == "Running")`,
		"filter reads a missing field": `$(context.data.status.ready)`,
	}

	for name, filter := range testcases {
		t.Run(name, func(t *testing.T) {
			eh, err := NewFromConfig(context.Background(), nil, v1alpha1.ActionMeta{Type: "unused"}, filter, nil, Options{})
			require.NoError(t, err)
			assert.EqualError(t, eh("resource-watcher", "update", object()), "event is filtered out")
		})
	}
}

func TestNewFromConfigRefusesBadExpressions(t *testing.T) {
	shell := v1alpha1.ActionMeta{Type: "task", Properties: &runtime.RawExtension{Raw: []byte(`{"cmd":["sh","-c","echo $(date)"]}`)}}

	testcases := map[string]struct {
		action  v1alpha1.ActionMeta
		filter  string
		options Options
		errs    bool
	}{
		"a CEL filter that does not compile":                     {action: v1alpha1.ActionMeta{Type: "task"}, filter: `$(context.data ==)`, errs: true},
		"properties not read for $( ) while expressions are off": {action: shell, errs: false},
		"$(date) refused once expressions are on":                {action: shell, options: Options{CELExpressions: true}, errs: true},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			_, err := NewFromConfig(context.Background(), nil, tc.action, tc.filter, nil, tc.options)
			assert.Equal(t, tc.errs, err != nil, "error: %v", err)
		})
	}
}

func TestCELFilterKeepsEvents(t *testing.T) {
	// Read as CUE, this filter would not compile and the event would be dropped.
	cli := fake.NewClientBuilder().Build()
	eh, err := NewFromConfig(context.Background(), cli, v1alpha1.ActionMeta{Type: "unused"}, `$(context.data.status.phase == "Pending")`, nil, Options{})
	require.NoError(t, err)

	// Past the filter, the action has no definition in this cluster.
	err = eh("resource-watcher", "update", object())
	require.Error(t, err)
	assert.NotEqual(t, "event is filtered out", err.Error())
}
