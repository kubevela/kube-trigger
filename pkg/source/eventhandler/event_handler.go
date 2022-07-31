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

package eventhandler

import (
	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/sirupsen/logrus"
)

// EventHandler is given to Source to be called.
type EventHandler func(sourceType string, event interface{})

// NewWithActionExecutor return a new EventHandler that will add a Job which will
// execute a given Action to Executor to be executed.
//
// For example, when the Source call this EventHandler, an Action will be executed.
func NewWithActionExecutor(exe *executor.Executor, reg *actionregistry.Registry, meta types.ActionMeta) EventHandler {
	return func(sourceType string, event interface{}) {
		j := executor.NewJob(reg, meta, sourceType, event)
		if j.Action == nil {
			return
		}
		logrus.WithField("eventhandler", "NewWithActionExecutor").
			Infof("new job %s created, adding to executor", j.Action.Type())
		ok := exe.AddJob(j)
		if !ok {
			logrus.Errorf("job queue full, cannot add action type %s", j.Action.Type())
		}
	}
}
