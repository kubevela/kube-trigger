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

package utils

import (
	"encoding/json"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	utilcue "github.com/kubevela/kube-trigger/pkg/util/cue"
)

// ValidateAndUnMarshal validates the input against the schema, and unmarshal
// the input cue to output. output must be a pointer.
func ValidateAndUnMarshal(schema string, input cue.Value, output interface{}) error {
	cueCtx := cuecontext.New()
	inputStr, err := utilcue.Marshal(input)
	if err != nil {
		return err
	}
	v := cueCtx.CompileString(schema + "\n" + inputStr)
	b, err := v.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, output)
}
