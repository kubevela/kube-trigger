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
	"github.com/kubevela/kube-trigger/pkg/filter/registry"
	"github.com/kubevela/kube-trigger/pkg/filter/types"
	"github.com/pkg/errors"
)

// ApplyFilters applies the given list of filters to an object. Since only filterMeta
// is given, it will try to fetch cached filters from registry if possible.
// All messages from filters will be returned as a list.
func ApplyFilters(
	event interface{},
	data interface{},
	filters []types.FilterMeta,
	reg *registry.Registry,
) (bool, []string, error) {
	var msgs []string
	for _, f := range filters {
		fInstance, err := reg.CreateOrGetInstance(f)
		if err != nil {
			return false, msgs, errors.Wrapf(err, "filter %s CreateOrGetInstance failed", f.Type)
		}
		kept, msg, err := fInstance.ApplyToObject(event, data)
		if err != nil {
			return false, msgs, errors.Wrapf(err, "error when applying filter %s to %v", f.Type, event)
		}
		if !kept {
			return false, msgs, nil
		}
		msgs = append(msgs, msg)
	}

	return true, msgs, nil
}
