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

package patchk8sobjects

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/imdario/mergo"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	utilcue "github.com/kubevela/kube-trigger/pkg/util/cue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	outputFieldName      = "output"
	dataFieldName        = "data"
	eventFieldName       = "event"
	patchTargetFieldName = "target"
)

const (
	// TypeName is the name of this action
	TypeName = "patch-k8s-objects"
)

// PatchK8sObjects patches k8s objects.
type PatchK8sObjects struct {
	c      client.Client
	prop   Properties
	logger *logrus.Entry
}

var _ types.Action = &PatchK8sObjects{}

func (pko *PatchK8sObjects) Run(ctx context.Context, sourceType string, event interface{}, data interface{}, _ []string) error {
	var contextStr string

	pko.logger.Infof("running, event souce: %s", sourceType)
	pko.logger.Debugf("running with event %v from %s", event, sourceType)

	// Add context.data using data.
	jsonByte, err := json.Marshal(data)
	if err == nil {
		pko.logger.Debugf("added context.%s: %s", dataFieldName, string(jsonByte))
		contextStr += fmt.Sprintf("context:{%s:%s}\n", dataFieldName, string(jsonByte))
	}

	// Add context.event using event.
	jsonByte, err = json.Marshal(event)
	if err == nil {
		pko.logger.Debugf("added context.%s: %s", eventFieldName, string(jsonByte))
		contextStr += fmt.Sprintf("context:{%s:%s}\n", eventFieldName, string(jsonByte))
	}

	gv, err := schema.ParseGroupVersion(pko.prop.PatchTarget.APIVersion)
	if err != nil {
		return err
	}

	gvk := gv.WithKind(pko.prop.PatchTarget.Kind)

	uList := unstructured.UnstructuredList{}
	uList.SetGroupVersionKind(gvk)

	var listOptions []client.ListOption
	if pko.prop.PatchTarget.Namespace != "" {
		listOptions = append(listOptions, client.InNamespace(pko.prop.PatchTarget.Namespace))
	}
	if len(pko.prop.PatchTarget.LabelSelectors) > 0 {
		selector := client.MatchingLabels{}
		for k, v := range pko.prop.PatchTarget.LabelSelectors {
			selector[k] = v
		}
		listOptions = append(listOptions, selector)
	}

	err = pko.c.List(ctx, &uList, listOptions...)
	if err != nil {
		pko.logger.Errorf("cannot list %v", pko.prop.PatchTarget)
		return err
	}

	var objList []unstructured.Unstructured

	for _, un := range uList.Items {
		// Do name filtering.
		targetName := pko.prop.PatchTarget.Name
		if targetName != "" && un.GetName() != targetName {
			continue
		}
		objList = append(objList, un)
	}

	pko.logger.Infof("found %d patch targets: %s", len(objList), gvk)

	// Patch each one.
	for i, un := range objList {
		pko.logger.Infof("patching target %d: %s", i+1, un.GetName())
		err = pko.updateObject(ctx, contextStr, un)
		if err != nil {
			return err
		}
		pko.logger.Infof("target %d patched: %s", i+1, un.GetName())
	}

	pko.logger.Info("action finished successfully")

	return nil
}

func (pko *PatchK8sObjects) updateObject(ctx context.Context, contextStr string, un unstructured.Unstructured) error {
	jsonByte, err := json.Marshal(un.Object)
	if err == nil {
		pko.logger.Debugf("added context.%s: %s", patchTargetFieldName, string(jsonByte))
		contextStr += fmt.Sprintf("context:{%s:%s}\n", patchTargetFieldName, string(jsonByte))
	}

	cueCtx := cuecontext.New()
	// context+patch string
	// We can use a string builder to go faster, but I didn't bother here.
	// This should not affect much.
	v := cueCtx.CompileString(pko.prop.Patch + "\n" + contextStr)
	if v.Err() != nil {
		return v.Err()
	}

	// Get evaluated patch.
	vPatch := v.LookupPath(cue.ParsePath(outputFieldName)).Eval()
	if vPatch.Err() != nil {
		return errors.Wrapf(vPatch.Err(), "did you forget to put `output` field inside your patch?")
	}

	// Put patch into a map.
	patchOut := make(map[string]interface{})
	err = utilcue.UnMarshal(vPatch, patchOut)
	if err != nil {
		return err
	}

	pko.logger.Debugf("parsed patch, going to apply: %v", patchOut)

	// Merge patch with object.
	err = mergo.Merge(&un.Object, patchOut, mergo.WithOverride)
	if err != nil {
		return err
	}

	pko.logger.Debugf("merged with patch, ready to update: %v", un.Object)

	// Apply merged object.
	return pko.c.Update(ctx, &un)
}

func (pko *PatchK8sObjects) Init(c types.Common, properties map[string]interface{}) error {
	var err error
	pko.logger = logrus.WithField("action", TypeName)
	pko.prop = Properties{}
	// Parse properties.
	err = pko.prop.parse(properties)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties")
	}
	pko.logger.Debugf("parsed propertise: %v", pko.prop)
	pko.c = c.Client
	pko.logger.Debugf("initialized")
	return nil
}

func (pko *PatchK8sObjects) Validate(properties map[string]interface{}) error {
	p := &Properties{}
	return p.parse(properties)
}

func (pko *PatchK8sObjects) Type() string {
	return TypeName
}

func (pko *PatchK8sObjects) New() types.Action {
	return &PatchK8sObjects{}
}

func (pko *PatchK8sObjects) AllowConcurrency() bool {
	return pko.prop.AllowConcurrency
}

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//go:generate ../../../../hack/generate-go-const-from-file.sh properties.cue propertiesCUETemplate properties

//+kubebuilder:object:generate=true
type Properties struct {
	PatchTarget PatchTarget `json:"patchTarget"`
	Patch       string      `json:"patch"`
	//+optional
	AllowConcurrency bool `json:"allowConcurrency"`
}

//+kubebuilder:object:generate=true
type PatchTarget struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	//+optional
	Namespace string `json:"namespace"`
	//+optional
	Name string `json:"name"`
	//+optional
	LabelSelectors map[string]string `json:"labelSelectors"`
}

// parse parses, evaluate, validate, and apply defaults.
func (p *Properties) parse(prop map[string]interface{}) error {
	return utilcue.ValidateAndUnMarshal(propertiesCUETemplate, prop, p)
}
