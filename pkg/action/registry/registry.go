package registry

import (
	"fmt"
	"sync"

	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/utils"
)

type Registry struct {
	reg  map[string]types.Action
	lock sync.RWMutex
}

func NewWithBuiltinActions() *Registry {
	ret := New()
	RegisterBuiltinActions(ret)
	return ret
}

func New() *Registry {
	r := Registry{}
	r.reg = make(map[string]types.Action)
	r.lock = sync.RWMutex{}
	return &r
}

func (r *Registry) ResetToBuiltinOnes() {
	newReg := NewWithBuiltinActions()
	r.reg = newReg.reg
}

func (r *Registry) RegisterExistingInstance(meta types.ActionMeta, instance types.Action) error {
	if meta.Raw == "" {
		return fmt.Errorf("filter meta raw info is empty")
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Raw] = instance
	return nil
}
func (r *Registry) CreateOrGetInstance(meta types.ActionMeta) (types.Action, error) {
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
		c, err := utils.GetClient()
		if err != nil {
			return nil, err
		}
		err = newInstance.Init(types.Common{Client: *c}, meta.Properties)
		if err != nil {
			return nil, err
		}
		_ = r.RegisterExistingInstance(meta, newInstance)
		return newInstance, nil
	}
	return instance, nil
}

func (r *Registry) GetInstance(meta types.ActionMeta) (types.Action, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Raw]
	return f, ok
}

func (r *Registry) RegisterType(meta types.ActionMeta, initial types.Action) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.reg[meta.Type] = initial
}

func (r *Registry) GetType(meta types.ActionMeta) (types.Action, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	f, ok := r.reg[meta.Type]
	return f, ok
}
