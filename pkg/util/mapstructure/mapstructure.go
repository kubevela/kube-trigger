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

package mapstructure

import (
	"encoding/json"

	"github.com/mitchellh/mapstructure"
)

// Decode unmarshalls a map[string]interface{} into a structure.
func Decode(in map[string]interface{}, out interface{}) error {
	return mapstructure.Decode(in, out)
}

// Encode marshals a structure into a map[string]interface{}.
func Encode(in interface{}, out *map[string]interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
