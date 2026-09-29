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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildContext(t *testing.T) {
	data := map[string]interface{}{"generation": int64(2)}
	changed := map[string]interface{}{"generation": int64(2)}

	testcases := map[string]struct {
		payload Payload
		want    []string
	}{
		"update carries changed": {
			payload: Payload{SourceType: "resource-watcher", Event: "event", Data: data, Changed: changed},
			want:    []string{"sourceType", "event", "data", "timestamp", "changed"},
		},
		"create and delete have no changed": {
			payload: Payload{SourceType: "resource-watcher", Event: "event", Data: data},
			want:    []string{"sourceType", "event", "data", "timestamp"},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			ctx := buildContext(tc.payload)
			var keys []string
			for k := range ctx {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, tc.want, keys)
			assert.Equal(t, "resource-watcher", ctx["sourceType"])
			assert.Equal(t, "event", ctx["event"])
			assert.Equal(t, data, ctx["data"])
			if tc.payload.Changed != nil {
				assert.Equal(t, changed, ctx["changed"])
			}
		})
	}
}
