/*
Copyright 2023 The KubeVela Authors.

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
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNewFromFileOrDir(t *testing.T) {
	a := assert.New(t)
	// From file
	cf, err := NewFromFileOrDir("testdata/golden/conf.cue")
	a.NoError(err)
	a.Equal(1, len(cf.Triggers))
	// From dir
	cd, err := NewFromFileOrDir("testdata/golden")
	a.NoError(err)
	a.Equal(4, len(cd.Triggers))
	// Every trigger should be the same
	prev := cf.Triggers[0]
	for _, t := range cd.Triggers {
		reflect.DeepEqual(prev, t)
	}
}

func TestNewFromFileOrDirInvalid(t *testing.T) {
	a := assert.New(t)

	// Report error if file is invalid
	_, err := NewFromFileOrDir("testdata/invalidext/conf.invalid")
	a.Error(err)

	// No error if files in dir is invalid, just skips
	c, err := NewFromFileOrDir("testdata/invalidext")
	a.NoError(err)
	a.Equal(0, len(c.Triggers))
}
