/*
Copyright 2022 The KubeVela Authors.

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

package filter

import (
	"context"
	"testing"

	"github.com/kubevela/pkg/cue/cuex"
	"github.com/kubevela/pkg/util/stringtools"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Filters here use no external CUE packages, so compile without a cluster.
	cuex.EnableExternalPackageForDefaultCompiler = false
}

func TestBuildFilterTemplate(t *testing.T) {
	testcases := map[string]struct {
		src    string
		result string
	}{
		"build filter with a single expression": {
			src:    "a == 1",
			result: "filter: a == 1",
		},
		"build filter with import declarations": {
			src: `
			import "strings"
			strings.Contains("abc", "a")`,
			result: `
			import "strings"
			filter: {
				strings.Contains("abc", "a")
			}`,
		},
	}

	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			result, err := BuildFilterTemplate(testcase.src)
			assert.NoError(t, err)
			assert.Equal(t,
				stringtools.TrimLeadingIndent(testcase.result),
				stringtools.TrimLeadingIndent(result),
			)
		})
	}
}

func TestApplyFilterWithChanged(t *testing.T) {
	const specChanged = `context.changed.spec != _|_`

	testcases := map[string]struct {
		context map[string]interface{}
		kept    bool
	}{
		"create has no changed": {
			context: map[string]interface{}{"data": map[string]interface{}{}},
			kept:    false,
		},
		"update that changed the spec": {
			context: map[string]interface{}{"changed": map[string]interface{}{"spec": map[string]interface{}{"replicas": 3}}},
			kept:    true,
		},
		"update that only changed status": {
			context: map[string]interface{}{"changed": map[string]interface{}{"status": map[string]interface{}{"readyReplicas": 3}}},
			kept:    false,
		},
		"update that changed neither": {
			context: map[string]interface{}{"changed": map[string]interface{}{}},
			kept:    false,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			kept, err := ApplyFilter(context.Background(), tc.context, specChanged)
			assert.NoError(t, err)
			assert.Equal(t, tc.kept, kept)
		})
	}
}

func TestApplyFilterWithChangedLabels(t *testing.T) {
	const labelsChanged = `context.changed.metadata.labels != _|_`
	changedLabels := func(labels interface{}) map[string]interface{} {
		return map[string]interface{}{"changed": map[string]interface{}{"metadata": map[string]interface{}{"labels": labels}}}
	}

	testcases := map[string]struct {
		context map[string]interface{}
		kept    bool
	}{
		"label changed": {
			context: changedLabels(map[string]interface{}{"tier": "api"}),
			kept:    true,
		},
		"only label removed, so the map is null": {
			context: changedLabels(nil),
			kept:    true,
		},
		"labels untouched": {
			context: map[string]interface{}{"changed": map[string]interface{}{"spec": map[string]interface{}{"replicas": 3}}},
			kept:    false,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			kept, err := ApplyFilter(context.Background(), tc.context, labelsChanged)
			assert.NoError(t, err)
			assert.Equal(t, tc.kept, kept)
		})
	}
}

func TestApplyFilterChangedKeepsIntegers(t *testing.T) {
	ctx := map[string]interface{}{"changed": map[string]interface{}{"spec": map[string]interface{}{"replicas": int64(3)}}}
	kept, err := ApplyFilter(context.Background(), ctx, `(int & context.changed.spec.replicas) == 3`)
	assert.NoError(t, err)
	assert.True(t, kept)
}

func TestApplyFilterChangedToAValue(t *testing.T) {
	const becameRunning = `(context.changed.status.phase & "Running") != _|_`
	phase := func(p string) map[string]interface{} {
		return map[string]interface{}{"changed": map[string]interface{}{"status": map[string]interface{}{"phase": p}}}
	}

	testcases := map[string]struct {
		context map[string]interface{}
		kept    bool
	}{
		"phase became Running":  {context: phase("Running"), kept: true},
		"phase became Pending":  {context: phase("Pending"), kept: false},
		"phase untouched":       {context: map[string]interface{}{"changed": map[string]interface{}{"spec": map[string]interface{}{}}}, kept: false},
		"create has no changed": {context: map[string]interface{}{"data": map[string]interface{}{}}, kept: false},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			kept, err := ApplyFilter(context.Background(), tc.context, becameRunning)
			assert.NoError(t, err)
			assert.Equal(t, tc.kept, kept)
		})
	}
}
