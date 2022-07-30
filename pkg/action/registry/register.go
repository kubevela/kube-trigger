package registry

import (
	"github.com/kubevela/kube-trigger/pkg/action/builtin/updatek8sobject"
	"github.com/kubevela/kube-trigger/pkg/action/types"
)

func RegisterBuiltinActions(reg *Registry) {

	uko := &updatek8sobject.UpdateK8sObject{}
	ukoMeta := types.ActionMeta{
		Type: uko.Type(),
	}
	reg.RegisterType(ukoMeta, uko)

}
