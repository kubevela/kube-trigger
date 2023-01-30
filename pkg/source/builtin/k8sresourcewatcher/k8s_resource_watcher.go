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

package k8sresourcewatcher

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/controller"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
)

type K8sResourceWatcher struct {
	configs       map[string]*types.Config
	eventHandlers map[string][]eventhandler.EventHandler
	logger        *logrus.Entry
}

var _ sourcetypes.Source = &K8sResourceWatcher{}

func (w *K8sResourceWatcher) New() sourcetypes.Source {
	return &K8sResourceWatcher{
		configs:       make(map[string]*types.Config),
		eventHandlers: make(map[string][]eventhandler.EventHandler),
	}
}

func (w *K8sResourceWatcher) Parse(properties *runtime.RawExtension) (*types.Config, error) {
	props, err := properties.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ctrlConf := &types.Config{}
	err = json.Unmarshal(props, ctrlConf)
	if err != nil {
		return nil, errors.Wrapf(err, "error when parsing properties for %s", w.Type())
	}
	return ctrlConf, nil
}

func (w *K8sResourceWatcher) Init(properties *runtime.RawExtension, eh eventhandler.EventHandler) error {
	var err error

	props, err := properties.MarshalJSON()
	if err != nil {
		return err
	}
	ctrlConf := &types.Config{}
	err = json.Unmarshal(props, ctrlConf)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties for %s", w.Type())
	}
	if orig, ok := w.configs[ctrlConf.Key()]; ok {
		orig.Merge(*ctrlConf)
		w.configs[ctrlConf.Key()] = orig
	} else {
		w.configs[ctrlConf.Key()] = ctrlConf
	}
	w.eventHandlers[ctrlConf.Key()] = append(w.eventHandlers[ctrlConf.Key()], eh)

	w.logger = logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher)

	w.logger.Debugf("initialized")
	return nil
}

func (w *K8sResourceWatcher) Run(ctx context.Context) error {
	for k, config := range w.configs {
		go func(c *types.Config, handlers []eventhandler.EventHandler) {
			resourceController := controller.Setup(*c, handlers)
			resourceController.Run(ctx.Done())
		}(config, w.eventHandlers[k])
	}
	return nil
}

func (w *K8sResourceWatcher) Type() string {
	return v1alpha1.SourceTypeResourceWatcher
}
