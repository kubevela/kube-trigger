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
	"context"
	"fmt"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	"github.com/kubevela/kube-trigger/pkg/templates"
)

// Config is what actually stores configs in memory.
// When marshalling or unmarshalling, simplified.ConfigWrapper will be used instead to
// make it easier for the user to write.
type Config struct {
	Triggers []v1alpha1.TriggerMeta `json:"triggers"`
}

// Validate validates config.
func (c *Config) Validate(ctx context.Context, sourceReg *sourceregistry.Registry) error {
	if len(c.Triggers) == 0 {
		return fmt.Errorf("no triggers found")
	}
	// TODO(charlie0129): gather all errors before returning
	for _, w := range c.Triggers {
		if _, ok := sourceReg.Get(w.Source.Type); !ok {
			return fmt.Errorf("no such source found: %s", w.Source.Type)
		}
		if _, err := templates.NewLoader("action").LoadTemplate(ctx, w.Action.Type); err != nil {
			return fmt.Errorf("no such action found: %s", w.Action.Type)
		}
	}

	return nil
}
