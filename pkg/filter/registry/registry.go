package registry

import (
	"fmt"
	"sync"

	"github.com/kubevela/kube-trigger/pkg/filter/types"
)

// Registry stores all instantiated filters to improve performance.
type Registry struct {
	reg  map[string]types.Filter
	lock sync.RWMutex
}

func NewWithBuiltinFilters() *Registry {
	ret := New()
	RegisterBuiltinFilters(ret)
	return ret
}

func New() *Registry {
	r := Registry{}
	r.reg = make(map[string]types.Filter)
	r.lock = sync.RWMutex{}
	return &r
}

func (r *Registry) ResetToBuiltinOnes() {
	newReg := NewWithBuiltinFilters()
	r.reg = newReg.reg
}

func (r *Registry) RegisterExistingInstance(meta types.FilterMeta, instance types.Filter) error {
	if meta.Raw == "" {
		return fmt.Errorf("filter meta raw info is empty")
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Raw] = instance
	return nil
}

func (r *Registry) CreateOrGetInstance(meta types.FilterMeta) (types.Filter, error) {
	if meta.Raw == "" {
		return nil, fmt.Errorf("filter meta raw info is empty")
	}
	initial, ok := r.GetType(meta)
	if !ok {
		return nil, fmt.Errorf("%s type %s does not exist", "filter", meta.Type)
	}
	instance, ok := r.GetInstance(meta)
	if !ok {
		newInstance := initial.New()
		err := newInstance.Init(meta.Properties)
		if err != nil {
			return nil, err
		}
		_ = r.RegisterExistingInstance(meta, newInstance)
		return newInstance, nil
	}
	return instance, nil
}

func (r *Registry) GetInstance(meta types.FilterMeta) (types.Filter, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Raw]
	return f, ok
}

func (r *Registry) RegisterType(meta types.FilterMeta, initial types.Filter) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Type] = initial
}

func (r *Registry) GetType(meta types.FilterMeta) (types.Filter, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Type]
	return f, ok
}