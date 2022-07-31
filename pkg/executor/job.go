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

package executor

import (
	"context"

	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/sirupsen/logrus"
)

// Job is an Action to be executed by the workers in the Executor.
type Job struct {
	Action     types.Action
	sourceType string
	event      interface{}
}

// NewJob creates a new job. It will fetch cached Action instance from Registry
// using provided ActionMeta. sourceType and event will be passed to the Action.Run
// method.
//
// See pkg/source/eventhandler.NewStoreWithActionExecutors() to get an idea about
// how it is used.
func NewJob(reg *actionregistry.Registry, meta types.ActionMeta, sourceType string, event interface{}) Job {
	var err error
	ret := Job{
		sourceType: sourceType,
		event:      event,
	}
	ret.Action, err = reg.CreateOrGetInstance(meta)
	if err != nil {
		logrus.Errorf("action type %s not found or cannot create instance: %s", meta.Type, err)
		return Job{}
	}
	return ret
}

// Run runs this job.
func (j *Job) Run(ctx context.Context) error {
	if j.Action == nil {
		return nil
	}
	return j.Action.Run(ctx, j.sourceType, j.event)
}
