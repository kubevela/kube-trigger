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

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	cases := map[string]struct {
		in  string
		out string
	}{
		"case with regexp": {
			in: `
foo: =~"*"
bar: 123
`,
			out: `{
	foo: =~"*"
	bar: 123
}`,
		},
	}
	r := require.New(t)
	for k, v := range cases {
		c := cuecontext.New()
		val := c.CompileString(v.in)
		str, err := Marshal(val)
		r.NoError(err, k)
		r.Equal(str, v.out, k)
	}
}

func TestUnMarshal(t *testing.T) {
	cases := map[string]struct {
		in  string
		out map[string]interface{}
		err bool
	}{
		"incomplete value": {
			in: `
foo: =~"*"
bar: 123
`,
			out: map[string]interface{}{},
			err: true,
		},
		"complete value": {
			in: `
foo: =~"."
bar: "123"
foo: "3"
`,
			out: map[string]interface{}{
				"foo": "3",
				"bar": "123",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			cc := cuecontext.New()
			val := cc.CompileString(c.in)
			dst := make(map[string]interface{})
			err := UnMarshal(val, dst)
			if c.err {
				r.Error(err)
			} else {
				r.NoError(err)
				r.Equal(dst, c.out)
			}
		})
	}
}
