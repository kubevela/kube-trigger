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

import utilcue "github.com/kubevela/kube-trigger/pkg/util/cue"

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//go:generate ../../../../../hack/generate-go-const-from-file.sh properties.cue propertiesCUETemplate properties

type Config struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Namespace  string   `json:"namespace"`
	Events     []string `json:"events"`
}

// Parse parses, evaluate, validate, and apply defaults.
func (c *Config) Parse(conf map[string]interface{}) error {
	return utilcue.ValidateAndUnMarshal(propertiesCUETemplate, conf, c)
}
