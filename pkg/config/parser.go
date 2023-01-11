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
	"os"
	"path/filepath"

	"cuelang.org/go/cue/cuecontext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var logger = logrus.WithField("config", "parser")

var parsers = map[string]func([]byte) (*Config, error){
	".cue":  cueParser,
	".yaml": yamlParser,
	".yml":  yamlParser,
	".json": jsonParser,
}

var (
	// ErrUnsupportedExtension is returned when the file extension is not supported.
	ErrUnsupportedExtension = errors.New("extension not supported")
)

// New news a config
func New() *Config {
	return &Config{}
}

// NewFromFileOrDir news a config from file or dir.
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
	fs, err := os.ReadDir(dir)
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

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read config file content")
	}

	return parser(data)
}

func cueParser(data []byte) (*Config, error) {
	c := cuecontext.New()
	v := c.CompileString(string(data))
	jsonByte, err := v.MarshalJSON()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = json.Unmarshal(jsonByte, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func jsonParser(data []byte) (*Config, error) {
	conf := &Config{}
	err := json.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func yamlParser(data []byte) (*Config, error) {
	conf := &Config{}
	err := yaml.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
