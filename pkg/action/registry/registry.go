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
	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/util/client"
)

// Registry stores Actions, both uninitialized and cached ones.
type Registry struct {
	lock        sync.Mutex
	reg         sync.Map
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
	r.reg = sync.Map{}
	r.lock = sync.Mutex{}
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
func (r *Registry) NewInstance(meta v1alpha1.ActionMeta) (types.Action, error) {
	initial, ok := r.GetTemplate(meta)
	if !ok {
		return nil, fmt.Errorf("action %s does not exist", meta.Template)
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
	return newInstance, err
}

// RegisterTemplate registers an uninitialized one.
func (r *Registry) RegisterTemplate(meta v1alpha1.ActionMeta, initial types.Action) {
	r.reg.Store(meta.Template, initial)
}

// GetTemplate gets an uninitialized one.
func (r *Registry) GetTemplate(meta v1alpha1.ActionMeta) (types.Action, bool) {
	f, ok := r.reg.Load(meta.Template)
	if !ok {
		return nil, ok
	}
	a, ok := f.(types.Action)
	return a, ok
}
