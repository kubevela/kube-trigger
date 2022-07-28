package cuevalidator

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/kubevela/kube-trigger/pkg/filter/builtin"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type CUEValidator struct {
	r *cue.Runtime
	v *cue.Value
	c *gocodec.Codec
}

func (c *CUEValidator) parseProperties(properties map[string]interface{}) (*Properties, error) {
	ret := &Properties{}

	err := mapstructure.Decode(properties, ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *CUEValidator) ApplyToObject(obj metav1.Object) (bool, error) {
	var err error
	u := unstructured.Unstructured{}
	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return false, err
	}
	err = c.c.Validate(*c.v, obj)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *CUEValidator) Init(properties map[string]interface{}) error {
	prop, err := c.parseProperties(properties)
	if err != nil {
		return err
	}

	c.r = &cue.Runtime{}

	instance, err := c.r.Compile("validator", prop.CUE.Template.String())
	if err != nil {
		return err
	}
	v := instance.Value()
	c.v = &v

	c.c = gocodec.New(c.r, nil)

	return nil
}

func (c *CUEValidator) Type() string {
	return "cue-validator"
}

func (c *CUEValidator) New() builtin.Filter {
	return &CUEValidator{}
}
