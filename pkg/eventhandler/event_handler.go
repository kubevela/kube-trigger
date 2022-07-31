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
	"fmt"

	"github.com/kubevela/kube-trigger/pkg/action/job"
	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	actiontypes "github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/executor"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
	filterutils "github.com/kubevela/kube-trigger/pkg/filter/utils"
	"github.com/sirupsen/logrus"
)

// EventHandler is given to Source to be called.
type EventHandler func(sourceType string, event interface{}) error

func New() EventHandler {
	return func(sourceType string, event interface{}) error {
		return nil
	}
}

func (e EventHandler) WithFilters(filters []filtertypes.FilterMeta, reg *filterregistry.Registry) EventHandler {
	return func(sourceType string, event interface{}) error {
		kept, err := filterutils.ApplyFilters(event, filters, reg)
		if err != nil {
			// TODO(charlie0129): log something
			return err
		}
		if !kept {
			return fmt.Errorf("filtered out")
		}
		return e(sourceType, event)
	}
}

func (e EventHandler) WithActions(exe *executor.Executor, actions []actiontypes.ActionMeta, reg *actionregistry.Registry) EventHandler {
	return func(sourceType string, event interface{}) error {
		err := e(sourceType, event)
		if err != nil {
			return err
		}
		for _, act := range actions {
			newJob, err := job.New(reg, act, sourceType, event)
			if err != nil {
				logrus.Errorf("error creating new job: %s", err)
				continue
			}
			_ = exe.AddJob(newJob)
		}
		return nil
	}
}
