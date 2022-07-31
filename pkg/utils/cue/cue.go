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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"k8s.io/apimachinery/pkg/util/json"
)

// Marshal marshals cue into string.
func Marshal(v cue.Value) (string, error) {
	syn := v.Syntax(cue.Raw())
	bs, err := format.Node(syn)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

// UnMarshal unmarshals cue value into a map. dst must be a pointer.
func UnMarshal(v cue.Value, dst map[string]interface{}) error {
	jsonByte, err := v.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonByte, &dst)
	if err != nil {
		return err
	}
	return nil
}
