package registry

import (
	"fmt"

	"github.com/kubevela/kube-trigger/pkg/filter/builtin"
)

var TypeRegistry Registry
var CachedRegistry Registry

type Registry struct {
	reg map[string]builtin.Filter
}

func (r *Registry) Register(filterType string, instance builtin.Filter) error {
	if filterType == "" {
		return fmt.Errorf("filter type is empty")
	}
	if r.reg == nil {
		r.reg = make(map[string]builtin.Filter)
	}

	r.reg[filterType] = instance
	return nil
}

func (r *Registry) Find(filterType string) (builtin.Filter, bool) {
	f, ok := r.reg[filterType]
	return f, ok
}
