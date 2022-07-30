package registry

import (
	"sync"

	"github.com/kubevela/kube-trigger/pkg/source/types"
)

type Registry struct {
	reg  map[string]types.Source
	lock sync.RWMutex
}

func NewWithBuiltinSources() *Registry {
	ret := New()
	RegisterBuiltinSources(ret)
	return ret
}

func New() *Registry {
	r := Registry{}
	r.reg = make(map[string]types.Source)
	r.lock = sync.RWMutex{}
	return &r
}

func (r *Registry) RegisterType(meta types.SourceMeta, initial types.Source) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Type] = initial
}

func (r *Registry) GetType(meta types.SourceMeta) (types.Source, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Type]
	return f, ok
}
