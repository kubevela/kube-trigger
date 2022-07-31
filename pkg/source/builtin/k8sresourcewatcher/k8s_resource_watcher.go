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
	"fmt"

	"cuelang.org/go/cue"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/config"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/controller"
	krwtypes "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type K8sResourceWatcher struct {
	resourceController *controller.Controller
	logger             *logrus.Entry
}

func (w *K8sResourceWatcher) New() sourcetypes.Source {
	return &K8sResourceWatcher{}
}

func (w *K8sResourceWatcher) Init(properties cue.Value, eh eventhandler.EventHandler) error {
	var err error

	ctrlConf := &config.Config{}
	err = ctrlConf.Parse(properties)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties for %s", w.Type())
	}

	w.resourceController = controller.Setup(*ctrlConf, eh)

	w.logger = logrus.WithField("source", krwtypes.TypeName)

	w.logger.Debugf("initialized")
	return nil
}

func (w *K8sResourceWatcher) Run(ctx context.Context) error {
	if w.resourceController == nil {
		return fmt.Errorf("controller has not been setup")
	}
	w.logger.Debugf("running")
	w.resourceController.Run(ctx.Done())
	return nil
}

func (w *K8sResourceWatcher) Type() string {
	return krwtypes.TypeName
}
