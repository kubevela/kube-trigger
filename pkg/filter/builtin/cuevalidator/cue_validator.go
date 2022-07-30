package cuevalidator

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/kubevela/kube-trigger/pkg/filter/types"
	utilscue "github.com/kubevela/kube-trigger/pkg/utils/cue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type CUEValidator struct {
	r      *cue.Runtime
	v      *cue.Value
	c      *gocodec.Codec
	logger *logrus.Entry
}

const (
	typeName = "cue-validator"
)

func (c *CUEValidator) parseProperties(properties cue.Value) (Properties, error) {
	// TODO(charlie0129): use a CUE to validate properties and provide default values
	v := properties.LookupPath(cue.ParsePath(TemplateFieldName))
	if v.Err() != nil {
		return Properties{}, v.Err()
	}
	return Properties{Template: v}, nil
}

func (c *CUEValidator) ApplyToObject(obj metav1.Object) (bool, error) {
	var err error

	u := unstructured.Unstructured{}
	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return false, err
	}

	c.logger.Debugf("applying to object name %s", u.GetName())

	err = c.c.Validate(*c.v, obj)
	if err != nil {
		c.logger.Debugf("object with name %s filtered out", u.GetName())
		return false, nil
	}

	c.logger.Debugf("object with name %s kept", u.GetName())
	return true, nil
}

func (c *CUEValidator) Init(properties cue.Value) error {
	prop, err := c.parseProperties(properties)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties")
	}

	c.r = &cue.Runtime{}

	cueStr, err := utilscue.Marshal(prop.Template)
	if err != nil {
		return err
	}

	instance, err := c.r.Compile("validator", cueStr)
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
