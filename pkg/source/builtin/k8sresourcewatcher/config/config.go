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

package config

import (
	"encoding/json"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	utilcue "github.com/kubevela/kube-trigger/pkg/util/cue"
)

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//go:generate ../../../../../hack/generate-properties-const-from-cue.sh properties.cue

type Config struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Namespace  string   `json:"namespace"`
	Events     []string `json:"events"`
}

// Parse parses, evaluate, validate, and apply defaults.
func (c *Config) Parse(vConf cue.Value) error {
	str, err := utilcue.Marshal(vConf)
	if err != nil {
		return err
	}

	cueCtx := cuecontext.New()
	v := cueCtx.CompileString(propertiesCUETemplate + str)

	js, err := v.MarshalJSON()
	if err != nil {
		return err
	}

	err = json.Unmarshal(js, c)
	if err != nil {
		return err
	}

	return nil
}
