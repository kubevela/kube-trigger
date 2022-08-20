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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	actiontype "github.com/kubevela/kube-trigger/pkg/action/types"
	filtertype "github.com/kubevela/kube-trigger/pkg/filter/types"
	sourcetype "github.com/kubevela/kube-trigger/pkg/source/types"
	utilcue "github.com/kubevela/kube-trigger/pkg/util/cue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("config", "parser")

var allowedExtensions = []string{
	"cue",
}

func New() *Config {
	return &Config{}
}

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
		logger.Debugf("files in dir: %v", files)
		for _, f := range files {
			if !isExtensionAllowed(f) {
				logger.Debugf("file %s does not have an acceptable extension", f)
				continue
			}
			subConfig := &Config{}
			err := subConfig.ParseFromFile(f)
			if err != nil {
				return nil, errors.Wrapf(err, "reading %s failed", f)
			}
			logger.Infof("loaded config from %s", f)
			c.Watchers = append(c.Watchers, subConfig.Watchers...)
		}
	} else {
		err := c.ParseFromFile(path)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func isExtensionAllowed(filename string) bool {
	var allowed bool
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(filename, "."+ext) {
			allowed = true
			break
		}
	}
	return allowed
}

func findFilesInDir(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (c *Config) Parse(confStr string) error {
	var err error

	cueCtx := cuecontext.New()
	vConf := cueCtx.CompileString(confStr)

	err = vConf.Validate()
	if err != nil {
		return err
	}

	vWatches := vConf.LookupPath(cue.ParsePath(WatchesFieldName))
	if vWatches.Err() != nil {
		return vWatches.Err()
	}

	c.Watchers, err = parseWatchers(vWatches)
	if err != nil {
		return err
	}

	logger.Debugf("configuration parsed: %v", c.Watchers)

	return nil
}

func (c *Config) ParseFromFile(path string) error {
	if !strings.HasSuffix(path, ".cue") {
		return fmt.Errorf("config files shoule be CUE source")
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("cannot load config %s. You should specify where your config resides.", path)
		return err
	}

	return c.Parse(string(data))
}
func parseWatchers(vWatches cue.Value) ([]WatchMeta, error) {
	var ret []WatchMeta

	vWatchList, err := vWatches.List()
	if err != nil {
		return nil, err
	}
	for i := 0; vWatchList.Next(); i++ {
		//nolint:govet // this err-shadowing fine
		watch, err := parseWatcher(vWatchList.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "error when parsing %s[%d]", WatchesFieldName, i)
		}
		ret = append(ret, watch)
	}

	return ret, nil
}

func parseWatcher(vWatch cue.Value) (WatchMeta, error) {
	var err error
	ret := WatchMeta{}

	vSource := vWatch.LookupPath(cue.ParsePath(SourceFieldName))
	if vSource.Err() != nil {
		return ret, vSource.Err()
	}

	ret.Source, err = parseSource(vSource)
	if err != nil {
		return ret, err
	}

	logger.Debugf("parsed source: %v", vSource)

	vFilters := vWatch.LookupPath(cue.ParsePath(FiltersFieldName))
	if vFilters.Err() != nil {
		return ret, vFilters.Err()
	}

	ret.Filters, err = parseFilters(vFilters)
	if err != nil {
		return ret, err
	}

	vActions := vWatch.LookupPath(cue.ParsePath(ActionsFieldName))
	if vActions.Err() != nil {
		return ret, vActions.Err()
	}

	ret.Actions, err = parseActions(vActions)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func parseSource(vSource cue.Value) (sourcetype.SourceMeta, error) {
	var err error
	ret := sourcetype.SourceMeta{}

	vType := vSource.LookupPath(cue.ParsePath(sourcetype.TypeFieldName))
	if vType.Err() != nil {
		return ret, vType.Err()
	}
	vProperties := vSource.LookupPath(cue.ParsePath(sourcetype.PropertiesFieldName))
	if vProperties.Err() != nil {
		return ret, vProperties.Err()
	}

	ret.Type, err = vType.String()
	if err != nil {
		return ret, err
	}
	ret.Properties = vProperties

	logger.Debugf("parsed source: %s", ret.Type)

	return ret, nil
}

func parseFilters(vFilters cue.Value) ([]filtertype.FilterMeta, error) {
	var ret []filtertype.FilterMeta

	vFilterList, err := vFilters.List()
	if err != nil {
		return nil, err
	}
	for i := 0; vFilterList.Next(); i++ {
		//nolint:govet // this err-shadowing fine
		filter, err := parseFilter(vFilterList.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "error when parsing %s[%d]", FiltersFieldName, i)
		}
		ret = append(ret, filter)
	}

	return ret, nil
}

func parseFilter(vFilter cue.Value) (filtertype.FilterMeta, error) {
	var err error
	ret := filtertype.FilterMeta{}

	vType := vFilter.LookupPath(cue.ParsePath(filtertype.TypeFieldName))
	if vType.Err() != nil {
		return ret, vType.Err()
	}
	vProperties := vFilter.LookupPath(cue.ParsePath(filtertype.PropertiesFieldName))
	if vProperties.Err() != nil {
		return ret, vProperties.Err()
	}

	ret.Type, err = vType.String()
	if err != nil {
		return ret, err
	}
	rawStr, err := utilcue.Marshal(vFilter)
	if err != nil {
		return ret, err
	}
	ret.Raw = rawStr
	ret.Properties = vProperties

	logger.Debugf("parsed filter: %s", ret.Raw)

	return ret, nil
}

func parseActions(vActions cue.Value) ([]actiontype.ActionMeta, error) {
	var ret []actiontype.ActionMeta

	vActionsList, err := vActions.List()
	if err != nil {
		return nil, err
	}
	for i := 0; vActionsList.Next(); i++ {
		//nolint:govet // this err-shadowing fine
		action, err := parseAction(vActionsList.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "error when parsing %s[%d]", ActionsFieldName, i)
		}
		ret = append(ret, action)
	}

	return ret, nil
}

func parseAction(vAction cue.Value) (actiontype.ActionMeta, error) {
	var err error
	ret := actiontype.ActionMeta{}

	vType := vAction.LookupPath(cue.ParsePath(actiontype.TypeFieldName))
	if vType.Err() != nil {
		return ret, vType.Err()
	}
	vProperties := vAction.LookupPath(cue.ParsePath(actiontype.PropertiesFieldName))
	if vProperties.Err() != nil {
		return ret, vProperties.Err()
	}

	ret.Type, err = vType.String()
	if err != nil {
		return ret, err
	}
	rawStr, err := utilcue.Marshal(vAction)
	if err != nil {
		return ret, err
	}
	ret.Raw = rawStr
	ret.Properties = vProperties

	logger.Debugf("parsed action: %s", ret.Raw)

	return ret, nil
}
