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
	"sync"

	"github.com/kubevela/kube-trigger/pkg/source/types"
)

// Registry stores builtin Sources.
type Registry struct {
	reg  map[string]types.Source
	lock sync.RWMutex
}

// NewWithBuiltinSources return a Registry with builtin sources registered.
// Ready for use.
func NewWithBuiltinSources() *Registry {
	ret := New()
	RegisterBuiltinSources(ret)
	return ret
}

// New creates a new Registry.
func New() *Registry {
	r := Registry{}
	r.reg = make(map[string]types.Source)
	r.lock = sync.RWMutex{}
	return &r
}

// Register registers a Source.
func (r *Registry) Register(meta types.SourceMeta, initial types.Source) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Type] = initial
}

// Get gets a Souce.
func (r *Registry) Get(meta types.SourceMeta) (types.Source, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Type]
	return f, ok
}
