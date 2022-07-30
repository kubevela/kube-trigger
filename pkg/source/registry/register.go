package registry

import (
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher"
	"github.com/kubevela/kube-trigger/pkg/source/types"
)

func RegisterBuiltinSources(reg *Registry) {

	krc := &k8sresourcewatcher.K8sResourceWatcher{}
	krcMeta := types.SourceMeta{
		Type: krc.Type(),
	}
	reg.RegisterType(krcMeta, krc)

}
