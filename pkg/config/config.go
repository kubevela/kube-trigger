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
	"fmt"

	"github.com/pkg/errors"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
)

// Config is what actually stores configs in memory.
// When marshalling or unmarshalling, simplified.ConfigWrapper will be used instead to
// make it easier for the user to write.
type Config struct {
	Triggers []v1alpha1.TriggerMeta `json:"triggers"`
}

// Validate validates config.
func (c *Config) Validate(
	sourceReg *sourceregistry.Registry,
	filterReg *filterregistry.Registry,
	actionReg *actionregistry.Registry,
) error {
	// TODO(charlie0129): gather all errors before returning
	for _, w := range c.Triggers {
		s, ok := sourceReg.Get(w.Source.Type)
		if !ok {
			return fmt.Errorf("no such source found: %s", w.Source.Type)
		}
		err := s.Validate(w.Source.Properties)
		if err != nil {
			return errors.Wrapf(err, "cannot validate source %s", w.Source.Type)
		}
		for _, a := range w.Actions {
			s, ok := actionReg.GetType(a)
			if !ok {
				return fmt.Errorf("no such action found: %s", w.Source.Type)
			}
			err := s.Validate(a.Properties)
			if err != nil {
				return errors.Wrapf(err, "cannot validate action %s", w.Source.Type)
			}
		}
		for _, f := range w.Filters {
			s, ok := filterReg.GetType(f)
			if !ok {
				return fmt.Errorf("no such filter found: %s", w.Source.Type)
			}
			err := s.Validate(f.Properties)
			if err != nil {
				return errors.Wrapf(err, "cannot validate filter %s", w.Source.Type)
			}
		}
	}

	return nil
}
