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
	"github.com/kubevela/kube-trigger/pkg/source/builtin/cronjob"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher"
	"github.com/kubevela/kube-trigger/pkg/source/types"
)

// RegisterBuiltinSources register builtin sources.
func RegisterBuiltinSources(reg *Registry) {
	registerFromInstance(reg, &k8sresourcewatcher.K8sResourceWatcher{})
	registerFromInstance(reg, &cronjob.CronJob{})
}

func registerFromInstance(reg *Registry, act types.Source) {
	ins := act
	insMeta := types.SourceMeta{
		Type: ins.Type(),
	}
	reg.Register(insMeta, ins)
}
