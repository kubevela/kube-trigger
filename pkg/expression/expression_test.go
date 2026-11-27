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

package expression

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// triggerContext is what the event handler gives filters and actions.
func triggerContext(changed map[string]interface{}) map[string]interface{} {
	data := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "web", "labels": map[string]interface{}{"app": "shop"}},
		"spec":       map[string]interface{}{"replicas": int64(3)},
		"status":     map[string]interface{}{"phase": "Running"},
	}}
	c := map[string]interface{}{
		"sourceType": "resource-watcher",
		"event":      map[string]interface{}{"type": "update", "cluster": "local"},
		"data":       data,
		"timestamp":  "2026-10-15T12:00:00Z",
	}
	if changed != nil {
		c["changed"] = changed
	}
	return c
}

func TestNewFilterTellsCELFromCUE(t *testing.T) {
	testcases := map[string]struct {
		filter string
		isCEL  bool
	}{
		"a single expression is CEL":       {filter: `$(context.data.status.phase == "Running")`, isCEL: true},
		"surrounding space is allowed":     {filter: "  $(true)\n", isCEL: true},
		"a CUE filter":                     {filter: `context.data.status.phase == "Running"`, isCEL: false},
		"$( inside a CUE string stays CUE": {filter: `context.data.metadata.name == "$(x)-web"`, isCEL: false},
		"an empty filter":                  {filter: "", isCEL: false},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			_, isCEL, err := NewFilter(tc.filter)
			require.NoError(t, err)
			assert.Equal(t, tc.isCEL, isCEL)
		})
	}
}

func TestFilterEval(t *testing.T) {
	testcases := map[string]struct {
		filter  string
		changed map[string]interface{}
		kept    bool
		errs    bool
	}{
		"true":                          {filter: `$(context.data.status.phase == "Running")`, kept: true},
		"false":                         {filter: `$(context.data.spec.replicas > 5)`, kept: false},
		"integers compare as integers":  {filter: `$(context.data.spec.replicas == 3)`, kept: true},
		"has() on a missing field":      {filter: `$(has(context.changed) && has(context.changed.spec))`, kept: false},
		"&& short-circuits":             {filter: `$(has(context.changed) && context.changed.spec.replicas > 1)`, kept: false},
		"reads changed when present":    {filter: `$(has(context.changed.spec))`, changed: map[string]interface{}{"spec": map[string]interface{}{"replicas": int64(4)}}, kept: true},
		"a missing field is an error":   {filter: `$(context.data.status.ready == true)`, errs: true},
		"a non-bool result is an error": {filter: `$(context.data.metadata.name)`, errs: true},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			f, isCEL, err := NewFilter(tc.filter)
			require.NoError(t, err)
			require.True(t, isCEL)
			kept, err := f.Eval(context.Background(), triggerContext(tc.changed))
			if tc.errs {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.kept, kept)
		})
	}
}

func TestNewFilterRefusesABadExpression(t *testing.T) {
	for name, filter := range map[string]string{
		"does not compile":      `$(context.data.status.phase ==)`,
		"reads an unknown name": `$(date == "today")`,
		"is never closed":       `$(context.data.status.phase == "Running"`,
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := NewFilter(filter)
			assert.Error(t, err)
		})
	}
}

func TestPropertiesEval(t *testing.T) {
	testcases := map[string]struct {
		props string
		want  string
	}{
		"a whole expression keeps its type": {
			props: `{"replicas":"$(context.data.spec.replicas)"}`,
			want:  `{"replicas":3}`,
		},
		"an expression inside text is joined in": {
			props: `{"name":"$(context.data.metadata.name)-backup","labels":{"app":"$(context.data.metadata.labels.app)"}}`,
			want:  `{"labels":{"app":"shop"},"name":"web-backup"}`,
		},
		"$$( is a literal $(": {
			props: `{"cmd":["sh","-c","echo $$(date) for $(context.data.metadata.name)"]}`,
			want:  `{"cmd":["sh","-c","echo $(date) for web"]}`,
		},
		"no expressions leaves it unchanged": {
			props: `{"image":"busybox","cmd":["true"]}`,
			want:  `{"cmd":["true"],"image":"busybox"}`,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			p, err := NewProperties(&runtime.RawExtension{Raw: []byte(tc.props)})
			require.NoError(t, err)
			got, err := p.Eval(context.Background(), triggerContext(nil))
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(got.Raw))
		})
	}
}

func TestNewPropertiesRefusesAShellSubstitution(t *testing.T) {
	// What an existing task action may hold; it must be written $$(date) once
	// expressions are on.
	_, err := NewProperties(&runtime.RawExtension{Raw: []byte(`{"cmd":["sh","-c","echo $(date)"]}`)})
	assert.ErrorContains(t, err, "cmd")
	assert.ErrorContains(t, err, "$$(", "says how to write a literal $(")
}

func TestNilProperties(t *testing.T) {
	p, err := NewProperties(nil)
	require.NoError(t, err)
	got, err := p.Eval(context.Background(), triggerContext(nil))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFilterGuardsAMapKey(t *testing.T) {
	// The README's guard: has() tests one field, whose parent must be there; `in`
	// tests a map key.
	f, _, err := NewFilter(`$(has(context.data.metadata.labels) && "tier" in context.data.metadata.labels && context.data.metadata.labels["tier"] == "web")`)
	require.NoError(t, err)

	testcases := map[string]struct {
		metadata map[string]interface{}
		kept     bool
	}{
		"no labels":       {metadata: map[string]interface{}{"name": "a"}, kept: false},
		"labels, no tier": {metadata: map[string]interface{}{"name": "a", "labels": map[string]interface{}{"app": "x"}}, kept: false},
		"tier is web":     {metadata: map[string]interface{}{"name": "a", "labels": map[string]interface{}{"tier": "web"}}, kept: true},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			data := &unstructured.Unstructured{Object: map[string]interface{}{"metadata": tc.metadata}}
			kept, err := f.Eval(context.Background(), map[string]interface{}{"data": data})
			require.NoError(t, err)
			assert.Equal(t, tc.kept, kept)
		})
	}
}
