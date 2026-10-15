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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
)

func trigger(filter, props string) v1alpha1.TriggerMeta {
	t := v1alpha1.TriggerMeta{Filter: filter, Action: v1alpha1.ActionMeta{Type: "task"}}
	if props != "" {
		t.Action.Properties = &runtime.RawExtension{Raw: []byte(props)}
	}
	return t
}

func TestValidateExpressions(t *testing.T) {
	c := Config{Triggers: []v1alpha1.TriggerMeta{
		trigger(`$(context.data.status.phase == "Running")`, `{"name":"$(context.data.metadata.name)"}`),
		trigger(`$(context.data ==)`, ""),
		trigger(`context.data.status.phase == "Running"`, `{"cmd":["sh","-c","echo $(date)"]}`),
	}}

	// With expressions off, only CEL filters are read.
	err := c.ValidateExpressions(false)
	assert.ErrorContains(t, err, "trigger 1")
	assert.NotContains(t, err.Error(), "trigger 2")

	// With them on, properties are read too, and every fault is reported.
	err = c.ValidateExpressions(true)
	assert.ErrorContains(t, err, "trigger 1")
	assert.ErrorContains(t, err, "trigger 2")
	assert.NotContains(t, err.Error(), "trigger 0")

	assert.NoError(t, (&Config{Triggers: c.Triggers[:1]}).ValidateExpressions(true))
}

// TestShippedExamplesCompile loads every example, a TriggerService or a
// standalone config, and checks its expressions with them on, as they are by
// default.
func TestShippedExamplesCompile(t *testing.T) {
	files, err := filepath.Glob("../../examples/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			b, err := os.ReadFile(f)
			require.NoError(t, err)
			var doc struct {
				Triggers []v1alpha1.TriggerMeta `json:"triggers"`
				Spec     struct {
					Triggers []v1alpha1.TriggerMeta `json:"triggers"`
				} `json:"spec"`
			}
			require.NoError(t, yaml.Unmarshal(b, &doc))
			c := &Config{Triggers: append(doc.Triggers, doc.Spec.Triggers...)}
			assert.NoError(t, c.ValidateExpressions(true))
		})
	}
}
