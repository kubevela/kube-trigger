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

func (r *Registry) size() int {
	cnt := 0
	r.reg.Range(func(k, v interface{}) bool {
		cnt++
		return true
	})
	return cnt
}

// RegisterExistingInstance registers an existing initialized Filters instance to Registry.
func (r *Registry) RegisterExistingInstance(meta types.FilterMeta, instance types.Filter) error {
	if meta.Raw == "" {
		return fmt.Errorf("filter meta raw info is empty")
	}
	if r.size() >= r.maxCapacity {
		return fmt.Errorf("filter registry max capacity exceed %d", r.maxCapacity)
	}
	r.reg.Store(meta.Raw, instance)
	return nil
}

// CreateOrGetInstance create or get an instance from Registry.
//
// If there is no initialized instance, but have an uninitialized one available,
// it creates a new one, initialize it, register it, and return it.
//
// If there is an initialized one available, it returns it.
//
// If this type does not exist, it errors out.
func (r *Registry) CreateOrGetInstance(meta types.FilterMeta) (types.Filter, error) {
	if meta.Raw == "" {
		return nil, fmt.Errorf("filter meta raw info is empty")
	}
	// Try to find an initialized one.
	instance, ok := r.GetInstance(meta)
	if ok {
		return instance, nil
	}
	// No initialized ones available.
	// Get uninitialized one and initialize it.
	initial, ok := r.GetType(meta)
	if !ok {
		return nil, fmt.Errorf("filter %s does not exist", meta.Type)
	}
	newInstance := initial.New()
	err := newInstance.Init(meta.Properties)
	if err != nil {
		return nil, err
	}
	err = r.RegisterExistingInstance(meta, newInstance)
	return newInstance, err
}

// GetInstance gets initialized instance.
func (r *Registry) GetInstance(meta types.FilterMeta) (types.Filter, bool) {
	f, ok := r.reg.Load(meta.Raw)
	if !ok {
		return nil, ok
	}
	a, ok := f.(types.Filter)
	return a, ok
}

// RegisterType registers an uninitialized one.
func (r *Registry) RegisterType(meta types.FilterMeta, initial types.Filter) {
	r.reg.Store(meta.Type, initial)
}

// GetType gets an uninitialized one.
func (r *Registry) GetType(meta types.FilterMeta) (types.Filter, bool) {
	f, ok := r.reg.Load(meta.Type)
	if !ok {
		return nil, ok
	}
	a, ok := f.(types.Filter)
	return a, ok
}
