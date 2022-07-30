package registry

import (
	"github.com/kubevela/kube-trigger/pkg/action/builtin/updatek8sobjects"
	"github.com/kubevela/kube-trigger/pkg/action/types"
)

func RegisterBuiltinActions(reg *Registry) {

	uko := &updatek8sobjects.UpdateK8sObject{}
	ukoMeta := types.ActionMeta{
		Type: uko.Type(),
	}
	reg.RegisterType(ukoMeta, uko)

}
