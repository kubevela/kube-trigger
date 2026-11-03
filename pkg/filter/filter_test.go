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

func TestApplyFilter(t *testing.T) {
	// The filters here use no external CUE packages, so compile without a cluster.
	cuex.EnableExternalPackageForDefaultCompiler = false
	contextData := map[string]interface{}{
		"data": map[string]interface{}{
			"metadata": map[string]interface{}{"name": "this-will-trigger-update-1"},
		},
	}
	testcases := map[string]struct {
		filter string
		kept   bool
	}{
		"empty filter": {
			filter: "",
			kept:   true,
		},
		"expression is true": {
			filter: `context.data.metadata.name == "this-will-trigger-update-1"`,
			kept:   true,
		},
		"expression is false": {
			filter: `context.data.metadata.name == "other"`,
			kept:   false,
		},
		"context matches": {
			filter: `context: data: metadata: name: =~"this-will-trigger-update-.*"`,
			kept:   true,
		},
		"context does not match": {
			filter: `context: data: metadata: name: =~"other-.*"`,
			kept:   false,
		},
		"context does not match a concrete value": {
			filter: `context: data: metadata: name: "other"`,
			kept:   false,
		},
	}

	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			kept, err := ApplyFilter(context.Background(), contextData, testcase.filter)
			assert.NoError(t, err)
			assert.Equal(t, testcase.kept, kept)
		})
	}
}
