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

package cue

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateAndUnMarshal(t *testing.T) {
	type testType struct {
		A string `json:"a"`
	}
	cases := map[string]struct {
		schema   string
		in       *runtime.RawExtension
		out      testType
		expected testType
		err      bool
	}{
		"successful": {
			schema: `a: string`,
			in:     &runtime.RawExtension{Raw: []byte(`{"a": "abc"}`)},
			out: struct {
				A string `json:"a"`
			}{},
			expected: struct {
				A string `json:"a"`
			}{A: "abc"},
			err: false,
		},
		"cannot marshal input": {
			schema: `a: string`,
			in:     &runtime.RawExtension{Raw: []byte(`"a":b`)},
			out: struct {
				A string `json:"a"`
			}{},
			expected: testType{},
			err:      true,
		},
		"cannot validate input": {
			schema: `a: string`,
			in:     &runtime.RawExtension{Raw: []byte(`{"a": 1}`)},
			out: struct {
				A string `json:"a"`
			}{},
			expected: testType{},
			err:      true,
		},
		"invalid schema": {
			schema: `a: `,
			in:     &runtime.RawExtension{Raw: []byte(`{"a": 1}`)},
			out: struct {
				A string `json:"a"`
			}{},
			expected: testType{},
			err:      true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			err := ValidateAndUnMarshal(c.schema, c.in, &c.out)
			if c.err {
				r.Error(err)
			} else {
				r.NoError(err)
				r.Equal(c.expected, c.out)
			}
		})
	}
}
