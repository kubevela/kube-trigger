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
	"os"
	"testing"

	"cuelang.org/go/cue"
	"github.com/kubevela/pkg/cue/cuex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// render compiles a trigger action definition with properties, without
// resolving its provider calls.
func render(t *testing.T, file string, properties map[string]interface{}) cue.Value {
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
		cuex.WithExtraData("parameter", properties), cuex.WithExtraData("context", map[string]interface{}{}),
		cuex.DisableResolveProviderFunctions{})
	require.NoError(t, err)
	return v
}

func TestPatchResourceTarget(t *testing.T) {
	patch := map[string]interface{}{"type": "merge", "data": map[string]interface{}{"metadata": map[string]interface{}{"labels": map[string]interface{}{"x": "y"}}}}
	resource := func(fields map[string]interface{}) map[string]interface{} {
		r := map[string]interface{}{"apiVersion": "v1", "kind": "ConfigMap"}
		for k, v := range fields {
			r[k] = v
		}
		return map[string]interface{}{"resource": r, "patch": patch}
	}

	testcases := map[string]struct {
		properties      map[string]interface{}
		name, namespace string
	}{
		"name and namespace": {
			properties: resource(map[string]interface{}{"name": "my-cm", "namespace": "shop"}),
			name:       "my-cm", namespace: "shop",
		},
		"namespace defaults to default": {
			properties: resource(map[string]interface{}{"name": "my-cm"}),
			name:       "my-cm", namespace: "default",
		},
		"direct fields win over metadata": {
			properties: resource(map[string]interface{}{
				"name": "my-cm", "namespace": "shop",
				"metadata": map[string]interface{}{"name": "other", "namespace": "elsewhere"},
			}),
			name: "my-cm", namespace: "shop",
		},
		"under metadata": {
			properties: resource(map[string]interface{}{"metadata": map[string]interface{}{"name": "my-cm", "namespace": "shop"}}),
			name:       "my-cm", namespace: "shop",
		},
	}

	t.Run("no name is an error", func(t *testing.T) {
		v := render(t, "../../config/definition/patch-resource.yaml", resource(nil))
		assert.Error(t, v.LookupPath(cue.ParsePath("patchObject.$params")).Validate(cue.Concrete(true)))
	})

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			v := render(t, "../../config/definition/patch-resource.yaml", tc.properties)
			require.NoError(t, v.Err())
			md := v.LookupPath(cue.ParsePath("patchObject.$params.resource.metadata"))
			gotName, err := md.LookupPath(cue.ParsePath("name")).String()
			require.NoError(t, err)
			gotNamespace, err := md.LookupPath(cue.ParsePath("namespace")).String()
			require.NoError(t, err)
			assert.Equal(t, tc.name, gotName)
			assert.Equal(t, tc.namespace, gotNamespace)
		})
	}
}
