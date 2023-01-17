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

package job

import (
	"context"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/executor"
)

// Job is the type of executor job.
type Job struct {
	action     types.Action
	sourceType string
	event      interface{}
	data       interface{}
	msgs       []string
}

var _ executor.Job = &Job{}

// New creates a new job. It will fetch cached Action instance from Registry
// using provided ActionMeta. sourceType and event will be passed to the Action.Run
// method.
func New(
	reg *registry.Registry,
	meta v1alpha1.ActionMeta,
	sourceType string,
	event interface{},
	data interface{},
	msgs []string,
) (*Job, error) {
	var err error
	ret := Job{
		sourceType: sourceType,
		event:      event,
		data:       data,
		msgs:       msgs,
	}
	ret.action, err = reg.NewInstance(meta)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// Type return job type
func (j *Job) Type() string {
	if j.action == nil {
		return ""
	}
	return j.action.Type()
}

// Run execute action
func (j *Job) Run(ctx context.Context) error {
	if j.action == nil {
		return nil
	}
	return j.action.Run(ctx, j.sourceType, j.event, j.data, j.msgs)
}

// AllowConcurrency returns whether the job allows concurrency.
func (j *Job) AllowConcurrency() bool {
	if j.action == nil {
		return true
	}
	return j.action.AllowConcurrency()
}
