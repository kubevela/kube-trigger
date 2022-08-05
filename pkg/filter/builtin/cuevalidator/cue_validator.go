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

package cuevalidator

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/kubevela/kube-trigger/pkg/filter/types"
	utilscue "github.com/kubevela/kube-trigger/pkg/util/cue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
)

type CUEValidator struct {
	r       *cue.Runtime
	v       *cue.Value
	c       *gocodec.Codec
	tmplStr string
	logger  *logrus.Entry
}

var _ types.Filter = &CUEValidator{}

const (
	typeName          = "cue-validator"
	TemplateFieldName = "template"
)

type Properties struct {
	Template cue.Value
}

func (c *CUEValidator) parseProperties(properties cue.Value) (Properties, error) {
	v := properties.LookupPath(cue.ParsePath(TemplateFieldName))
	if v.Err() != nil {
		return Properties{}, v.Err()
	}
	return Properties{Template: v}, nil
}

func (c *CUEValidator) ApplyToObject(obj interface{}) (bool, error) {
	var err error

	c.logger.Debugf("applying to object %v", obj)

	// This validation method is faster. Filter out unneeded events first.
	// But this may not be enough. If c.v have a field that is not in
	// obj, this will still succeed. Because it is just making sure obj
	// satisfies the constraints defined by c.v.
	// We need to make sure obj have c.v as well later.
	err = c.c.Validate(*c.v, obj)
	if err != nil {
		c.logger.Debugf("object is filtered out by stage 1: %s", err)
		return false, nil
	}

	// Event is kept. Do further filter.
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return false, err
	}
	cueCtx := cuecontext.New()
	_, err = cueCtx.CompileString(c.tmplStr + "\n" + string(jsonBytes)).MarshalJSON()
	if err != nil {
		c.logger.Debugf("object is filtered out by stage 2: %s", err)
		return false, nil
	}

	c.logger.Debugf("object is kept")

	return true, nil
}

func (c *CUEValidator) Init(properties cue.Value) error {
	prop, err := c.parseProperties(properties)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties")
	}

	c.r = &cue.Runtime{}

	c.tmplStr, err = utilscue.Marshal(prop.Template)
	if err != nil {
		return err
	}

	instance, err := c.r.Compile("validator", c.tmplStr)
	if err != nil {
		return err
	}
	v := instance.Value()

	c.v = &v

	c.c = gocodec.New(c.r, nil)

	c.logger = logrus.WithField("filter", typeName)
	c.logger.Debugf("initialized")

	return nil
}

func (c *CUEValidator) Type() string {
	return typeName
}

func (c *CUEValidator) New() types.Filter {
	return &CUEValidator{}
}
