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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMergePatch(t *testing.T) {
	// labels nil leaves metadata.labels out, as the API server does for an empty map.
	deployment := func(replicas int64, ready int64, labels map[string]interface{}) *unstructured.Unstructured {
		metadata := map[string]interface{}{"name": "app"}
		if labels != nil {
			metadata["labels"] = labels
		}
		return &unstructured.Unstructured{Object: map[string]interface{}{
			"metadata": metadata,
			"spec":     map[string]interface{}{"replicas": replicas},
			"status":   map[string]interface{}{"readyReplicas": ready},
		}}
	}
	spec := func(spec map[string]interface{}) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{"spec": spec}}
	}
	tier := map[string]interface{}{"tier": "web"}

	testcases := map[string]struct {
		from, to *unstructured.Unstructured
		patch    map[string]interface{}
	}{
		"spec change keeps integers as integers": {
			from:  deployment(1, 1, tier),
			to:    deployment(3, 1, tier),
			patch: map[string]interface{}{"spec": map[string]interface{}{"replicas": int64(3)}},
		},
		"status-only change": {
			from:  deployment(3, 1, tier),
			to:    deployment(3, 3, tier),
			patch: map[string]interface{}{"status": map[string]interface{}{"readyReplicas": int64(3)}},
		},
		"removing the only label removes the labels map": {
			from:  deployment(1, 1, tier),
			to:    deployment(1, 1, nil),
			patch: map[string]interface{}{"metadata": map[string]interface{}{"labels": nil}},
		},
		"integers beyond float64 precision stay exact": {
			from:  spec(map[string]interface{}{"id": int64(9007199254740993)}),
			to:    spec(map[string]interface{}{"id": int64(9007199254740992)}),
			patch: map[string]interface{}{"spec": map[string]interface{}{"id": int64(9007199254740992)}},
		},
		"changed list appears whole": {
			from:  spec(map[string]interface{}{"ports": []interface{}{int64(80), int64(443)}}),
			to:    spec(map[string]interface{}{"ports": []interface{}{int64(80), int64(8443)}}),
			patch: map[string]interface{}{"spec": map[string]interface{}{"ports": []interface{}{int64(80), int64(8443)}}},
		},
		"unchanged object": {
			from:  deployment(1, 1, tier),
			to:    deployment(1, 1, tier),
			patch: map[string]interface{}{},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			patch, err := MergePatch(tc.from, tc.to)
			assert.NoError(t, err)
			assert.Equal(t, tc.patch, patch)
		})
	}
}

func TestMergePatchOfNonObject(t *testing.T) {
	_, err := MergePatch(make(chan int), &unstructured.Unstructured{})
	assert.Error(t, err)
}
