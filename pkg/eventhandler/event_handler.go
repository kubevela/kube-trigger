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
// event is what event happened, containing a brief event object. Do not include
// complex objects in it. For example, a k8s-resource-watcher Source may contain
// what event happened (create, update, delete) in it.
//
// data is the detailed event, for machines to process, e.g. passed to filters to
// do filtering . You may put complex objects in it. For example,
// a k8s-resource-watcher Source may contain the entire object that is changed
// in it.
type EventHandler func(sourceType string, event interface{}, data interface{}) error

type Config struct {
	Filters        []filtertypes.FilterMeta
	Actions        []actiontypes.ActionMeta
	FilterRegistry *filterregistry.Registry
	ActionRegistry *actionregistry.Registry
	Executor       *executor.Executor
}

// New create a new EventHandler that does nothing.
func New() EventHandler {
	return func(sourceType string, event interface{}, data interface{}) error {
		return nil
	}
}

func NewFromConfig(c Config) EventHandler {
	filterLogger := logrus.WithField("eventhandler", "applyfilters")
	actionLogger := logrus.WithField("eventhandler", "addactionjob")
	return func(sourceType string, event interface{}, data interface{}) error {
		// Apply filters
		kept, msgs, err := filterutils.ApplyFilters(event, data, c.Filters, c.FilterRegistry)
		if err != nil {
			filterLogger.Errorf("error when applying filters to event %v: %s", event, err)
			return err
		}
		if !kept {
			filterLogger.Debugf("event %v is filtered out", event)
			filterLogger.Infof("event is filtered out")
			return fmt.Errorf("event is filtered out")
		}
		filterLogger.Infof("event passed filters")
		filterLogger.Infof("filters left these messages: %v", msgs)

		// Run actions
		for _, act := range c.Actions {
			//nolint:govet // ignore
			newJob, err := job.New(c.ActionRegistry, act, sourceType, event, data, msgs)
			if err != nil {
				actionLogger.Errorf("error when creating new job: %s", err)
				continue
			}
			err = c.Executor.AddJob(newJob)
			if err != nil {
				actionLogger.Errorf("error when adding job to executor: %s", err)
			}
		}

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
