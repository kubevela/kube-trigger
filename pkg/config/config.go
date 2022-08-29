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
	"fmt"

	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	actiontype "github.com/kubevela/kube-trigger/pkg/action/types"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertype "github.com/kubevela/kube-trigger/pkg/filter/types"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	sourcetype "github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// Config is what actually stores configs in memory.
// When marshalling or unmarshalling, simplified.ConfigWrapper will be used instead to
// make it easier for the user to write.
type Config struct {
	Triggers []TriggerMeta `json:"triggers"`
}

type TriggerMeta struct {
	Source  sourcetype.SourceMeta   `json:"source"`
	Filters []filtertype.FilterMeta `json:"filters"`
	Actions []actiontype.ActionMeta `json:"actions"`
}

// MarshalJSON will encode Config as ConfigWrapper, which is an external Config
// format for the user to hand-write.
func (c *Config) MarshalJSON() ([]byte, error) {
	cfg := ConfigWrapper{}
	err := cfg.FromConfig(c)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cfg)
}

// UnmarshalJSON will decode ConfigWrapper to Config, which is an internal Config
// for machines to read.
func (c *Config) UnmarshalJSON(src []byte) error {
	cfg := ConfigWrapper{}
	err := json.Unmarshal(src, &cfg)
	if err != nil {
		return err
	}
	return cfg.ToConfig(c)
}

func (c *Config) Parse(jsonByte []byte) error {
	err := json.Unmarshal(jsonByte, c)
	if err != nil {
		return errors.Wrapf(err, "cannot unmarshal config")
	}

	var newTriggers []TriggerMeta
	// Insert Raw field
	for _, w := range c.Triggers {
		var newActions []actiontype.ActionMeta
		for _, a := range w.Actions {
			b, err := json.Marshal(a.Properties)
			if err != nil {
				return err
			}
			a.Raw = string(b)
			newActions = append(newActions, a)
		}
		w.Actions = newActions
		var newFilters []filtertype.FilterMeta
		for _, f := range w.Filters {
			b, err := json.Marshal(f.Properties)
			if err != nil {
				return err
			}
			f.Raw = string(b)
			newFilters = append(newFilters, f)
		}
		w.Filters = newFilters
		newTriggers = append(newTriggers, w)
	}
	c.Triggers = newTriggers

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		b, _ := yaml.Marshal(c)
		logger.Debugf("configuration parsed: \n%s", string(b))
	}

	return nil
}

//nolint:gocognit // .
func (c *Config) Validate(
	sourceReg *sourceregistry.Registry,
	filterReg *filterregistry.Registry,
	actionReg *actionregistry.Registry,
) error {
	// TODO(charlie0129): gather all errors before returning
	for _, w := range c.Triggers {
		s, ok := sourceReg.Get(w.Source)
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
