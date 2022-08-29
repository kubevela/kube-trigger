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

	"github.com/kubevela/kube-trigger/pkg/action/builtin/bumpapplicationrevision"
	"github.com/kubevela/kube-trigger/pkg/action/builtin/patchk8sobjects"
	actiontypes "github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/filter/builtin/cuevalidator"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
	k8sresourcewatcherconfig "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/config"
	k8sresourcewatchertypes "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/kubevela/kube-trigger/pkg/util/mapstructure"
	"github.com/pkg/errors"
)

//nolint:revive //.
// ConfigWrapper is what the user writes as config. It is a wrapper over Config.
// It is meant to make it easier for the user to write configs.
type ConfigWrapper struct {
	Triggers []TriggerMetaWrapper `json:"triggers"`
}

//+kubebuilder:object:generate=true
type TriggerMetaWrapper struct {
	SourceMetaWrapper `json:",inline"`
	Filters           []FilterMetaWrapper `json:"filters"`
	Actions           []ActionMetaWrapper `json:"actions"`
}

// TODO(charlie0129): use code generation instead of manually adding them.

//+kubebuilder:object:generate=true
type SourceMetaWrapper struct {
	K8sResourceWatcher *k8sresourcewatcherconfig.Properties `json:"k8s-resource-watcher,omitempty"`
}

//+kubebuilder:object:generate=true
type FilterMetaWrapper struct {
	CUEValidator *cuevalidator.Properties `json:"cue-validator,omitempty"`
}

//+kubebuilder:object:generate=true
type ActionMetaWrapper struct {
	BumpApplicationRevision *bumpapplicationrevision.Properties `json:"bump-application-revision,omitempty"`
	PatchK8sObjects         *patchk8sobjects.Properties         `json:"patch-k8s-objects,omitempty"`
}

//nolint:gocognit //.
// FromConfig converts Config to ConfigWrapper.
func (c *ConfigWrapper) FromConfig(cfg *Config) error {
	for _, trigger := range cfg.Triggers {
		triggerMeta := TriggerMetaWrapper{}

		// Source
		switch trigger.Source.Type {
		// TODO(charlie0129): use code generation instead of manually adding them.
		case k8sresourcewatchertypes.TypeName:
			p := &k8sresourcewatcherconfig.Properties{}
			err := mapstructure.Decode(trigger.Source.Properties, p)
			if err != nil {
				return errors.Wrapf(err, "error when ummarshalling %s", k8sresourcewatchertypes.TypeName)
			}
			triggerMeta.K8sResourceWatcher = p
		default:
			return fmt.Errorf("unsupported source type %s", trigger.Source.Type)
		}

		// Filters
		for _, filter := range trigger.Filters {
			filterMeta := FilterMetaWrapper{}

			switch filter.Type {
			case cuevalidator.TypeName:
				p := &cuevalidator.Properties{}
				err := mapstructure.Decode(filter.Properties, p)
				if err != nil {
					return errors.Wrapf(err, "error when ummarshalling %s", cuevalidator.TypeName)
				}
				filterMeta.CUEValidator = p
			default:
				return fmt.Errorf("unsupported filter type %s", filter.Type)
			}

			triggerMeta.Filters = append(triggerMeta.Filters, filterMeta)
		}

		// Actions
		for _, action := range trigger.Actions {
			actionMeta := ActionMetaWrapper{}

			switch action.Type {
			case bumpapplicationrevision.TypeName:
				p := &bumpapplicationrevision.Properties{}
				err := mapstructure.Decode(action.Properties, p)
				if err != nil {
					return errors.Wrapf(err, "error when ummarshalling %s", bumpapplicationrevision.TypeName)
				}
				actionMeta.BumpApplicationRevision = p
			case patchk8sobjects.TypeName:
				p := &patchk8sobjects.Properties{}
				err := mapstructure.Decode(action.Properties, p)
				if err != nil {
					return errors.Wrapf(err, "error when ummarshalling %s", patchk8sobjects.TypeName)
				}
				actionMeta.PatchK8sObjects = p

			default:
				return fmt.Errorf("unsupported action type %s", action.Type)
			}

			triggerMeta.Actions = append(triggerMeta.Actions, actionMeta)
		}

		c.Triggers = append(c.Triggers, triggerMeta)
	}

	return nil
}

//nolint:funlen,gocognit //.
// ToConfig converts ConfigWrapper to Config.
func (c *ConfigWrapper) ToConfig(cfg *Config) error {
	for _, t := range c.Triggers {
		nt := TriggerMeta{}

		// Parse Source
		foundSources := 0
		if t.K8sResourceWatcher != nil {
			ns := sourcetypes.SourceMeta{}
			ns.Type = k8sresourcewatchertypes.TypeName
			err := mapstructure.Encode(t.K8sResourceWatcher, &ns.Properties)
			if err != nil {
				return errors.Wrapf(err, "error when marshalling %s", ns.Type)
			}
			nt.Source = ns
			foundSources++
		}
		// Other Sources...
		if foundSources != 1 {
			return fmt.Errorf("expected only 1 Source, but found %d", foundSources)
		}

		// Parse Filters
		for _, f := range t.Filters {
			nf := filtertypes.FilterMeta{}
			foundFilters := 0
			if f.CUEValidator != nil {
				nf.Type = cuevalidator.TypeName
				err := mapstructure.Encode(f.CUEValidator, &nf.Properties)
				if err != nil {
					return errors.Wrapf(err, "error when marshalling %s", nf.Type)
				}
				b, err := json.Marshal(f.CUEValidator)
				if err != nil {
					return err
				}
				nf.Raw = string(b)
				nt.Filters = append(nt.Filters, nf)
				foundFilters++
			}
			// Other Filters...
			if foundFilters != 1 {
				return fmt.Errorf("expected only 1 Filter, but found %d", foundSources)
			}
		}

		// Parse Actions
		for _, a := range t.Actions {
			na := actiontypes.ActionMeta{}
			foundActions := 0
			if a.BumpApplicationRevision != nil {
				na.Type = bumpapplicationrevision.TypeName
				err := mapstructure.Encode(a.BumpApplicationRevision, &na.Properties)
				if err != nil {
					return errors.Wrapf(err, "error when marshalling %s", na.Type)
				}
				b, err := json.Marshal(a.BumpApplicationRevision)
				if err != nil {
					return err
				}
				na.Raw = string(b)
				nt.Actions = append(nt.Actions, na)
				foundActions++
			}
			if a.PatchK8sObjects != nil {
				na.Type = patchk8sobjects.TypeName
				err := mapstructure.Encode(a.PatchK8sObjects, &na.Properties)
				if err != nil {
					return errors.Wrapf(err, "error when marshalling %s", na.Type)
				}
				b, err := json.Marshal(a.PatchK8sObjects)
				if err != nil {
					return err
				}
				na.Raw = string(b)
				nt.Actions = append(nt.Actions, na)
				foundActions++
			}
			// Other Actions...
			if foundActions != 1 {
				return fmt.Errorf("expected only 1 Action, but found %d", foundSources)
			}
		}
		cfg.Triggers = append(cfg.Triggers, nt)
	}

	return nil
}
