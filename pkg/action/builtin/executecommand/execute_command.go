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

package executecommand

import (
	"context"
	"os/exec"

	"github.com/kubevela/kube-trigger/pkg/action/types"
	cue2 "github.com/kubevela/kube-trigger/pkg/util/cue"
)

const (
	typeName = "execute-command"
)

type ExecuteCommand struct {
	cmd exec.Cmd
}

var _ types.Action = &ExecuteCommand{}

func (ec *ExecuteCommand) Run(ctx context.Context, sourceType string, event interface{}, data interface{}, messages []string) error {

	return nil
}

func (ec *ExecuteCommand) Init(c types.Common, properties map[string]interface{}) error {

	return nil
}

func (ec *ExecuteCommand) Validate(properties map[string]interface{}) error {
	return nil
}

func (ec *ExecuteCommand) Type() string {
	return typeName
}

func (ec *ExecuteCommand) New() types.Action {
	return &ExecuteCommand{}
}

func (ec *ExecuteCommand) AllowConcurrency() bool {

	return false
}

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//go:generate ../../../../hack/generate-go-const-from-file.sh properties.cue propertiesCUETemplate properties

type Properties struct {
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	LabelSelectors map[string]string `json:"labelSelectors"`
}

func (p *Properties) parse(prop map[string]interface{}) error {
	return cue2.ValidateAndUnMarshal(propertiesCUETemplate, prop, p)
}
