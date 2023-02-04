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

package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubevela/pkg/cue/cuex"
	"github.com/mitchellh/hashstructure/v2"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/kubevela/kube-trigger/pkg/templates"
)

// Job is the type of executor job.
type Job struct {
	sourceType string
	id         string
	context    string
	properties string
	template   string
}

var _ executor.Job = &Job{}

// New creates a new job. It will fetch cached Action instance from Registry
// using provided ActionMeta. sourceType and event will be passed to the Action.Run
// method.
func New(meta v1alpha1.ActionMeta, contextData map[string]interface{}) (*Job, error) {
	var err error
	template, err := templates.NewLoader("action").LoadTemplate(context.Background(), meta.Type)
	if err != nil {
		return nil, err
	}
	contextByte, err := json.Marshal(contextData)
	if err != nil {
		return nil, err
	}
	propByte := []byte("{}")
	if meta.Properties != nil {
		propByte, err = meta.Properties.MarshalJSON()
		if err != nil {
			return nil, err
		}
	}
	id, err := computeHash(meta)
	if err != nil {
		return nil, err
	}
	ret := Job{
		id:         id,
		template:   template,
		sourceType: meta.Type,
		context:    fmt.Sprintf("context: %s", string(contextByte)),
		properties: fmt.Sprintf("parameter: %s", string(propByte)),
	}

	return &ret, nil
}

func computeHash(obj interface{}) (string, error) {
	// compute a hash value of any resource spec
	specHash, err := hashstructure.Hash(obj, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	specHashLabel := strconv.FormatUint(specHash, 16)
	return specHashLabel, nil
}

// Type return job type
func (j *Job) Type() string {
	return j.sourceType
}

// ID return job id
func (j *Job) ID() string {
	return j.id
}

// Run execute action
func (j *Job) Run(ctx context.Context) error {
	str := strings.Join([]string{j.template, j.properties, j.context}, "\n")
	v, err := cuex.CompileString(ctx, str)
	if err != nil {
		return err
	}
	if v.Err() != nil {
		return v.Err()
	}
	return nil
}

// AllowConcurrency returns whether the job allows concurrency.
func (j *Job) AllowConcurrency() bool {
	return false
}
