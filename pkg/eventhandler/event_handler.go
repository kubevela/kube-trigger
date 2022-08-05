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

// EventHandler is given to Source to be called. Source is responsible to call
// this function.
//
// sourceType is what type the Source is.
//
// event is what event happened, containing a brief event object. Normally, it
// is for humans to read, e.g. sending this to Telegram bot. So, do not include
// complex objects in it.
//
// data is the detailed event, for machines to process, e.g. passed to filters to
// do filtering . You may put complex objects in it. For example,
// a k8s-resource-watcher Source may contain the entire object in it.
type EventHandler func(sourceType string, event interface{}, data interface{}) error

// New create a new EventHandler that does nothing.
func New() EventHandler {
	return func(sourceType string, event interface{}, data interface{}) error {
		return nil
	}
}

// AddHandlerBefore adds a new EventHandler to be called before e is called.
func (e EventHandler) AddHandlerBefore(eh EventHandler) EventHandler {
	return func(sourceType string, event interface{}, data interface{}) error {
		err := eh(sourceType, event, data)
		if err != nil {
			return err
		}
		return e(sourceType, event, data)
	}
}

// AddHandlerAfter adds a new EventHandler to be called after e is called.
func (e EventHandler) AddHandlerAfter(eh EventHandler) EventHandler {
	return func(sourceType string, event interface{}, data interface{}) error {
		err := e(sourceType, event, data)
		if err != nil {
			return err
		}
		return eh(sourceType, event, data)
	}
}

// WithFilters applies filters to event before e is called.
func (e EventHandler) WithFilters(
	filters []filtertypes.FilterMeta,
	reg *filterregistry.Registry,
) EventHandler {
	logger := logrus.WithField("eventhandler", "applyfilters")
	return e.AddHandlerBefore(func(sourceType string, event interface{}, data interface{}) error {
		kept, err := filterutils.ApplyFilters(data, filters, reg)
		if err != nil {
			logger.Errorf("error when applying filters to event %v: %s", event, err)
			return err
		}
		if !kept {
			logger.Debugf("event %v is filtered out", event)
			logger.Infof("event is filtered out")
			return fmt.Errorf("event is filtered out")
		}
		logger.Infof("event passed filters")
		return nil
	})
}

// WithActions adds jobs that will execute actions to Executor after e is called.
func (e EventHandler) WithActions(
	exe *executor.Executor,
	actions []actiontypes.ActionMeta,
	reg *actionregistry.Registry,
) EventHandler {
	logger := logrus.WithField("eventhandler", "newactionjob")
	return e.AddHandlerAfter(func(sourceType string, event interface{}, data interface{}) error {
		for _, act := range actions {
			newJob, err := job.New(reg, act, sourceType, event, data)
			if err != nil {
				logger.Errorf("error when creating new job: %s", err)
				continue
			}
			err = exe.AddJob(newJob)
			if err != nil {
				logger.Errorf("error when adding job to executor: %s", err)
			}
		}
		return nil
	})
}
