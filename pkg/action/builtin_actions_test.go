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

package action

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"cuelang.org/go/cue"
	"github.com/kubevela/pkg/cue/cuex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// deploymentEvent is the context an action sees for an update of a Deployment.
func deploymentEvent() map[string]interface{} {
	return map[string]interface{}{
		"sourceType": "resource-watcher",
		"event":      map[string]interface{}{"type": "update", "cluster": "local"},
		"timestamp":  "2026-01-02T03:04:05Z",
		"data": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "web",
				"namespace": "shop",
				"uid":       "1234",
				"labels":    map[string]interface{}{"app": "web"},
			},
		},
	}
}

// renderAction compiles a built-in action as a job does, without resolving its
// provider calls, and returns the error a job would fail with.
func renderAction(t *testing.T, file string, properties, eventContext map[string]interface{}) (cue.Value, error) {
	cuex.EnableExternalPackageForDefaultCompiler = false
	b, err := os.ReadFile(file)
	require.NoError(t, err)
	var def struct {
		Spec struct {
			Templates map[string]string `json:"templates"`
		} `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(b, &def))
	v, err := cuex.DefaultCompiler.Get().CompileStringWithOptions(context.Background(), def.Spec.Templates["main.cue"],
		cuex.WithExtraData("parameter", properties), cuex.WithExtraData("context", eventContext),
		cuex.DisableResolveProviderFunctions{})
	if err != nil {
		return v, err
	}
	return v, v.Err()
}

// decode reads the value at path into out.
func decode(t *testing.T, v cue.Value, path string, out interface{}) {
	b, err := v.LookupPath(cue.ParsePath(path)).MarshalJSON()
	require.NoError(t, err, path)
	require.NoError(t, json.Unmarshal(b, out), path)
}

func TestRecordEventAppendsTheEvent(t *testing.T) {
	v, err := renderAction(t, "../../config/definition/record-event.yaml", map[string]interface{}{}, deploymentEvent())
	require.NoError(t, err)

	var events []map[string]interface{}
	decode(t, v, "patch.$params.patch.data.events", &events)
	require.Len(t, events, 1)
	assert.Equal(t, "update", events[0]["type"])
	assert.Equal(t, "2026-01-02T03:04:05Z", events[0]["timestamp"])
	assert.Equal(t, map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "name": "web", "namespace": "shop"}, events[0]["resource"])
}

func TestTaskPassesTheSourceAndExtraEnv(t *testing.T) {
	testcases := map[string]struct {
		env  []interface{}
		want []map[string]string
	}{
		"no env given": {
			want: []map[string]string{
				{"name": "SOURCE_NAME", "value": "web"},
				{"name": "SOURCE_NAMESPACE", "value": "shop"},
			},
		},
		"extra env given": {
			env: []interface{}{map[string]interface{}{"name": "COLOR", "value": "blue"}},
			want: []map[string]string{
				{"name": "SOURCE_NAME", "value": "web"},
				{"name": "SOURCE_NAMESPACE", "value": "shop"},
				{"name": "COLOR", "value": "blue"},
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			properties := map[string]interface{}{"image": "busybox", "cmd": []interface{}{"true"}}
			if tc.env != nil {
				properties["env"] = tc.env
			}
			v, err := renderAction(t, "../../config/definition/task.yaml", properties, deploymentEvent())
			require.NoError(t, err)

			var env []map[string]string
			decode(t, v, "apply.$params.resource.spec.template.spec.containers[0].env", &env)
			assert.Equal(t, tc.want, env)
		})
	}
}

// Every built-in action must at least compile for an ordinary event.
func TestBuiltinActionsCompile(t *testing.T) {
	testcases := map[string]map[string]interface{}{
		"bump-application-revision.yaml": {},
		"create-event-listener.yaml":     {},
		"patch-resource.yaml": {
			"resource": map[string]interface{}{"apiVersion": "v1", "kind": "ConfigMap", "name": "cm"},
			"patch":    map[string]interface{}{"type": "merge", "data": map[string]interface{}{"data": map[string]interface{}{"a": "b"}}},
		},
		"record-event.yaml": {},
		"task.yaml":         {"image": "busybox", "cmd": []interface{}{"true"}},
	}

	for file, properties := range testcases {
		t.Run(file, func(t *testing.T) {
			_, err := renderAction(t, "../../config/definition/"+file, properties, deploymentEvent())
			assert.NoError(t, err)
		})
	}
}
