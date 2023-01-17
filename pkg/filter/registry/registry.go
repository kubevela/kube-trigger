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

package registry

import (
	"fmt"
	"sync"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/filter/types"
)

// Registry stores Filters, both uninitialized and cached ones.
type Registry struct {
	reg         sync.Map
	maxCapacity int
}

// NewWithBuiltinFilters return a Registry with builtin filters registered.
// Ready for use.
func NewWithBuiltinFilters(capacity int) *Registry {
	ret := New(capacity)
	RegisterBuiltinFilters(ret)
	return ret
}

// New creates a new Registry.
func New(capacity int) *Registry {
	r := Registry{}
	r.reg = sync.Map{}
	r.maxCapacity = capacity
	return &r
}

// NewInstance new an instance from Registry.
//
// If there is no initialized instance, but have an uninitialized one available,
// it creates a new one, initialize it, register it, and return it.
//
// If there is an initialized one available, it returns it.
//
// If this type does not exist, it errors out.
func (r *Registry) NewInstance(meta v1alpha1.FilterMeta) (types.Filter, error) {
	initial, ok := r.GetType(meta)
	if !ok {
		return nil, fmt.Errorf("filter %s does not exist", meta.Type)
	}
	newInstance := initial.New()
	err := newInstance.Init(meta.Properties)
	return newInstance, err
}

// RegisterType registers an uninitialized one.
func (r *Registry) RegisterType(meta v1alpha1.FilterMeta, initial types.Filter) {
	r.reg.Store(meta.Type, initial)
}

// GetType gets an uninitialized one.
func (r *Registry) GetType(meta v1alpha1.FilterMeta) (types.Filter, bool) {
	f, ok := r.reg.Load(meta.Type)
	if !ok {
		return nil, ok
	}
	a, ok := f.(types.Filter)
	return a, ok
}
