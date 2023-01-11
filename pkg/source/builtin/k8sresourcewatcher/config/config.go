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
	utilcue "github.com/kubevela/kube-trigger/pkg/util/cue"
	"k8s.io/apimachinery/pkg/runtime"
)

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//
//go:generate ../../../../../hack/generate-go-const-from-file.sh properties.cue propertiesCUETemplate properties

type Properties struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	//+optional
	Namespace string `json:"namespace,omitempty"`
	//+optional
	Events         []string          `json:"events,omitempty"`
	MatchingLabels map[string]string `json:"matchingLabels,omitempty"`
}

// Parse parses, evaluate, validate, and apply defaults.
func (c *Properties) Parse(conf *runtime.RawExtension) error {
	return utilcue.ValidateAndUnMarshal(propertiesCUETemplate, conf, c)
}
