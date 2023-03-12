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
