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
	"io/ioutil"
	"os"
	"path/filepath"

	"cuelang.org/go/cue/cuecontext"
	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	actiontype "github.com/kubevela/kube-trigger/pkg/action/types"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertype "github.com/kubevela/kube-trigger/pkg/filter/types"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	sourcetype "github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Config struct {
	Triggers []TriggerMeta `json:"triggers"`
}

type TriggerMeta struct {
	Source  sourcetype.SourceMeta   `json:"source"`
	Filters []filtertype.FilterMeta `json:"filters"`
	Actions []actiontype.ActionMeta `json:"actions"`
}

var logger = logrus.WithField("config", "parser")

var parsers = map[string]func([]byte) (*Config, error){
	".cue":  CUEParser,
	".yaml": YAMLParser,
	".yml":  YAMLParser,
	".json": JSONParser,
}

var (
	ErrUnsupportedExtension = errors.New("extension not supported")
)

func New() *Config {
	return &Config{}
}

//nolint:nestif,govet // .
func NewFromFileOrDir(path string) (*Config, error) {
	c := &Config{}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		files, err := findFilesInDir(path)
		if err != nil {
			return nil, err
		}
		logger.Debugf("loading files: %v", files)
		for _, f := range files {
			subConfig, err := parseFromFile(f)
			if err != nil {
				if errors.Is(err, ErrUnsupportedExtension) {
					continue
				}
				return nil, errors.Wrapf(err, "reading %s failed", f)
			}
			logger.Infof("loaded config from %s", f)
			c.Triggers = append(c.Triggers, subConfig.Triggers...)
		}
	} else {
		c, err = parseFromFile(path)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func findFilesInDir(dir string) ([]string, error) {
	var files []string
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, f.Name()))
	}
	return files, err
}

func parseFromFile(path string) (*Config, error) {
	ext := filepath.Ext(path)
	parser, ok := parsers[ext]
	if !ok {
		logger.Warnf("file %s is skipped because extension %s is not supported", path, ext)
		return nil, ErrUnsupportedExtension
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read config file content")
	}

	return parser(data)
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

	logger.Debugf("configuration parsed: %v", c.Triggers)

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

func CUEParser(data []byte) (*Config, error) {
	c := cuecontext.New()
	v := c.CompileString(string(data))
	jsonByte, err := v.MarshalJSON()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = conf.Parse(jsonByte)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func JSONParser(data []byte) (*Config, error) {
	conf := &Config{}
	err := conf.Parse(data)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func YAMLParser(data []byte) (*Config, error) {
	jsonByte, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	err = conf.Parse(jsonByte)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
