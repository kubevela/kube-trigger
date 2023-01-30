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
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"github.com/kubevela/pkg/cue/cuex"
)

// ApplyFilter applies the given filter to an object.
func ApplyFilter(ctx context.Context, contextData map[string]interface{}, filter string) (bool, error) {
	contextByte, err := json.Marshal(contextData)
	if err != nil {
		return false, err
	}
	contextTemplate := fmt.Sprintf("context: %s", string(contextByte))
	compiler := cuex.DefaultCompiler.Get()
	filterVal, err := compiler.CompileString(ctx, fmt.Sprintf("filter: {\n%s\n%s\n}", filter, contextTemplate))
	if err != nil {
		return false, err
	}
	if filterVal.Err() != nil {
		return false, filterVal.Err()
	}
	result := filterVal.LookupPath(cue.ParsePath("filter"))
	if filterVal.LookupPath(cue.ParsePath("filter.filter")).Exists() {
		result = filterVal.LookupPath(cue.ParsePath("filter.filter"))
	}
	if result.Err() != nil {
		return false, result.Err()
	}
	resultBool, err := result.Bool()
	// if the result is not a bool, return true to pass the filter
	if err != nil {
		return true, nil
	}
	return resultBool, nil
}
