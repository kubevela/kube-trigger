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

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/action"
	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/kubevela/kube-trigger/pkg/filter"
)

// Payload is what a Source hands to its EventHandlers for one event. It is
// shared by every handler for that event and by the jobs they queue, and its
// objects may be informer cache entries, so treat it as read-only.
type Payload struct {
	// SourceType is what type the Source is.
	SourceType string
	// Event is what happened, as a brief object. Do not include complex objects
	// in it. For example, a resource-watcher Source puts the event type (create,
	// update, delete) in it.
	Event interface{}
	// Data is the detailed event, for machines to process, e.g. passed to
	// filters. For example, a resource-watcher Source puts the entire object
	// that changed in it.
	Data interface{}
	// Changed is what the event changed in Data, as a JSON merge patch, for
	// sources that know, e.g. the resource-watcher on an update. It is nil
	// otherwise.
	Changed map[string]interface{}
}

// EventHandler is given to Source to be called once per event. Source is
// responsible to call this function.
type EventHandler func(p Payload) error

// Config is the config for trigger
type Config struct {
	Handler  map[v1alpha1.ActionMeta]string
	Executor *executor.Executor
}

// New create a new EventHandler that does nothing.
func New() EventHandler {
	return func(_ Payload) error {
		return nil
	}
}

// NewFromConfig creates a new EventHandler from config.
func NewFromConfig(ctx context.Context, cli client.Client, actionMeta v1alpha1.ActionMeta, filterMeta string, executor *executor.Executor) EventHandler {
	filterLogger := logrus.WithField("eventhandler", "applyfilters")
	actionLogger := logrus.WithField("eventhandler", "addactionjob")
	return func(p Payload) error {
		// TODO: use handler to handle
		// Apply filters
		context := buildContext(p)
		kept, err := filter.ApplyFilter(ctx, context, filterMeta)
		if err != nil {
			filterLogger.Errorf("error when applying filters to event %v: %s", p.Event, err)
		}
		if !kept {
			filterLogger.Debugf("event %v is filtered out", p.Event)
			filterLogger.Infof("event is filtered out")
			return fmt.Errorf("event is filtered out")
		}
		filterLogger.Infof("event passed filters")

		// Run actions
		newJob, err := action.New(ctx, cli, actionMeta, context)
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
	}
}

// buildContext is the context given to filters and actions. It has changed
// only when the payload does, so CUE can test it with != _|_.
func buildContext(p Payload) map[string]interface{} {
	context := map[string]interface{}{
		"sourceType": p.SourceType,
		"event":      p.Event,
		"data":       p.Data,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	if p.Changed != nil {
		context["changed"] = p.Changed
	}
	return context
}

// AddHandlerBefore adds a new EventHandler to be called before e is called.
func (e EventHandler) AddHandlerBefore(eh EventHandler) EventHandler {
	return func(p Payload) error {
		err := eh(p)
		if err != nil {
			return err
		}
		return e(p)
	}
}

// AddHandlerAfter adds a new EventHandler to be called after e is called.
func (e EventHandler) AddHandlerAfter(eh EventHandler) EventHandler {
	return func(p Payload) error {
		err := e(p)
		if err != nil {
			return err
		}
		return eh(p)
	}
}
