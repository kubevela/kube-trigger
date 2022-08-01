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

	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/utils/client"
)

// Registry stores Actions, both uninitialized and cached ones.
type Registry struct {
	reg         map[string]types.Action
	lock        sync.RWMutex
	maxCapacity int
}

// NewWithBuiltinActions return a Registry with builtin actions registered.
// Ready for use.
func NewWithBuiltinActions(capacity int) *Registry {
	ret := New(capacity)
	RegisterBuiltinActions(ret)
	return ret
}

// New creates a new Registry.
func New(capacity int) *Registry {
	r := Registry{}
	r.reg = make(map[string]types.Action)
	r.lock = sync.RWMutex{}
	r.maxCapacity = capacity
	return &r
}

// RegisterExistingInstance registers an existing initialized Action instance to Registry.
func (r *Registry) RegisterExistingInstance(meta types.ActionMeta, instance types.Action) error {
	if meta.Raw == "" {
		return fmt.Errorf("action meta raw info is empty")
	}
	if len(r.reg) >= r.maxCapacity {
		return fmt.Errorf("action registry max capacity exceed %d", r.maxCapacity)
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Raw] = instance
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
func (r *Registry) CreateOrGetInstance(meta types.ActionMeta) (types.Action, error) {
	if meta.Raw == "" {
		return nil, fmt.Errorf("action meta raw info is empty")
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
		return nil, fmt.Errorf("action %s does not exist", meta.Type)
	}
	newInstance := initial.New()
	c, err := client.GetClient()
	if err != nil {
		return nil, err
	}
	err = newInstance.Init(types.Common{Client: *c}, meta.Properties)
	if err != nil {
		return nil, err
	}
	err = r.RegisterExistingInstance(meta, newInstance)
	return newInstance, err
}

// GetInstance gets initialized instance.
func (r *Registry) GetInstance(meta types.ActionMeta) (types.Action, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Raw]
	return f, ok
}

// RegisterType registers an uninitialized one.
func (r *Registry) RegisterType(meta types.ActionMeta, initial types.Action) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Type] = initial
}

// GetType gets an uninitialized one.
func (r *Registry) GetType(meta types.ActionMeta) (types.Action, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Type]
	return f, ok
}
