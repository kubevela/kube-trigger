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
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/action"
	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/kubevela/kube-trigger/pkg/expression"
	"github.com/kubevela/kube-trigger/pkg/filter"
)

// EventHandler is given to Source to be called. Source is responsible to call
// this function.
//
// sourceType is what type the Source is.
//
// event is what event happened, containing a brief event object. Do not include
// complex objects in it. For example, a resource-watcher Source may contain
// what event happened (create, update, delete) in it.
//
// data is the detailed event, for machines to process, e.g. passed to filters to
// do filtering . You may put complex objects in it. For example,
// a resource-watcher Source may contain the entire object that is changed
// in it.
type EventHandler func(sourceType string, event interface{}, data interface{}) error

// Config is the config for trigger
type Config struct {
	Handler  map[v1alpha1.ActionMeta]string
	Executor *executor.Executor
}

// New create a new EventHandler that does nothing.
func New() EventHandler {
	return func(_ string, _ interface{}, _ interface{}) error {
		return nil
	}
}

// Options adjusts how a trigger's event handler reads its filter and action.
type Options struct {
	// CELExpressions renders $( ) expressions in action properties, where a
	// literal $(, such as Kubernetes' own $(VAR), is written $$(. A filter
	// that is a single $( ) is CEL either way, as no CUE filter has that form.
	CELExpressions bool
}

// NewFromConfig creates a new EventHandler from config. Its CEL filter and
// action properties are planned here, so one that does not compile is an
// error now rather than on every event.
func NewFromConfig(ctx context.Context, cli client.Client, actionMeta v1alpha1.ActionMeta, filterMeta string, executor *executor.Executor, opts Options) (EventHandler, error) {
	filterLogger := logrus.WithField("eventhandler", "applyfilters")
	actionLogger := logrus.WithField("eventhandler", "addactionjob")
	celFilter, isCEL, err := expression.NewFilter(filterMeta)
	if err != nil {
		return nil, errors.WithMessage(err, "filter")
	}
	var props *expression.Properties
	if opts.CELExpressions {
		if props, err = expression.NewProperties(actionMeta.Properties); err != nil {
			return nil, errors.WithMessagef(err, "properties of action %s", actionMeta.Type)
		}
	}
	return func(sourceType string, event interface{}, data interface{}) error {
		// TODO: use handler to handle
		// Apply filters
		context := map[string]interface{}{
			"sourceType": sourceType,
			"event":      event,
			"data":       data,
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		var kept bool
		var err error
		if isCEL {
			kept, err = celFilter.Eval(ctx, context)
		} else {
			kept, err = filter.ApplyFilter(ctx, context, filterMeta)
		}
		if err != nil {
			filterLogger.Errorf("error when applying filters to event %v: %s", event, err)
		}
		if !kept {
			filterLogger.Debugf("event %v is filtered out", event)
			filterLogger.Infof("event is filtered out")
			return fmt.Errorf("event is filtered out")
		}
		filterLogger.Infof("event passed filters")

		// Run actions
		properties := actionMeta.Properties
		if props != nil {
			if properties, err = props.Eval(ctx, context); err != nil {
				actionLogger.Errorf("error when evaluating the properties of action %s: %s", actionMeta.Type, err)
				return err
			}
		}
		newJob, err := action.New(ctx, cli, actionMeta, properties, context)
		if err != nil {
			actionLogger.Errorf("error when creating new job: %s", err)
			return err
		}
		err = executor.AddJob(newJob)
		if err != nil {
			actionLogger.Errorf("error when adding job to executor: %s", err)
			return err
		}

		return nil
	}, nil
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
