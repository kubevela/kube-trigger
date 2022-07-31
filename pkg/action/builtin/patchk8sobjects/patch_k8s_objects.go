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
	krwevent "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/event"
	utilcue "github.com/kubevela/kube-trigger/pkg/utils/cue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	outputFieldName       = "output"
	sourceObjectFieldName = "sourceObject"
	patchTargetFieldName  = "patchTarget"
)

const (
	// typeName is the name of this action
	typeName = "patch-k8s-objects"
)

type PatchK8sObjects struct {
	c      client.Client
	prop   Properties
	logger *logrus.Entry
}

func (u *PatchK8sObjects) Run(ctx context.Context, sourceType string, event interface{}) error {
	var contextStr string

	u.logger.Infof("running, event souce: %s", sourceType)
	u.logger.Debugf("running with event %v from %s", event, sourceType)

	// Only if it is from k8s-resource-watcher, we add context.sourceObject
	e, ok := event.(krwevent.Event)
	if ok {
		jsonByte, err := json.Marshal(e.Obj)
		if err == nil {
			u.logger.Debugf("added context.%s: %s", sourceObjectFieldName, string(jsonByte))
			contextStr += fmt.Sprintf("context:{%s:%s}\n", sourceObjectFieldName, string(jsonByte))
		}
	} else {
		u.logger.Infof("event is not a k8s-resource-watcher event, so context.%s available", sourceObjectFieldName)
	}

	gv, err := schema.ParseGroupVersion(u.prop.PatchTarget.APIVersion)
	if err != nil {
		return err
	}

	gvk := gv.WithKind(u.prop.PatchTarget.Kind)

	unstructuredObjList := unstructured.UnstructuredList{}
	unstructuredObjList.SetGroupVersionKind(gvk)

	var listOptions []client.ListOption
	if u.prop.PatchTarget.Namespace != "" {
		listOptions = append(listOptions, client.InNamespace(u.prop.PatchTarget.Namespace))
	}
	if len(u.prop.PatchTarget.LabelSelectors) > 0 {
		selector := client.MatchingLabels{}
		for k, v := range u.prop.PatchTarget.LabelSelectors {
			selector[k] = v
		}
		listOptions = append(listOptions, selector)
	}

	err = u.c.List(ctx, &unstructuredObjList, listOptions...)
	if err != nil {
		u.logger.Errorf("cannot list %v", u.prop.PatchTarget)
		return err
	}

	var objList []unstructured.Unstructured

	for _, un := range unstructuredObjList.Items {
		// Do name filtering.
		targetName := u.prop.PatchTarget.Name
		if targetName != "" && un.GetName() != targetName {
			continue
		}
		objList = append(objList, un)
	}

	u.logger.Infof("found %d patch targets: %s", len(objList), gvk)

	// Patch each one.
	for i, un := range objList {
		u.logger.Infof("patching target %d: %s", i+1, un.GetName())
		err = u.updateObject(ctx, contextStr, un)
		if err != nil {
			return err
		}
		u.logger.Infof("target %d patched: %s", i+1, un.GetName())
	}

	u.logger.Info("action finished successfully")

	return nil
}

func (u *PatchK8sObjects) updateObject(ctx context.Context, contextStr string, un unstructured.Unstructured) error {
	jsonByte, err := json.Marshal(un.Object)
	if err == nil {
		u.logger.Debugf("added context.%s: %s", patchTargetFieldName, string(jsonByte))
		contextStr += fmt.Sprintf("context:{%s:%s}\n", patchTargetFieldName, string(jsonByte))
	}

	cueCtx := cuecontext.New()
	// context+patch string
	// We can use a string builder to go faster, but I didn't bother here.
	// This should not affect much.
	v := cueCtx.CompileString(u.prop.Patch + "\n" + contextStr)
	if v.Err() != nil {
		return v.Err()
	}

	// Get evaluated patch.
	vPatch := v.LookupPath(cue.ParsePath(outputFieldName)).Eval()
	if vPatch.Err() != nil {
		return vPatch.Err()
	}

	// Put patch into a map.
	patchOut := make(map[string]interface{})
	err = utilcue.UnMarshal(vPatch, patchOut)
	if err != nil {
		return err
	}

	u.logger.Debugf("parsed patch, going to apply: %v", patchOut)

	// Merge patch with object.
	err = mergo.Merge(&un.Object, patchOut, mergo.WithOverride)
	if err != nil {
		return err
	}

	u.logger.Debugf("merged with patch, ready to update: %v", un.Object)

	// Apply merged object.
	err = u.c.Update(ctx, &un)
	if err != nil {
		return err
	}

	return nil
}

func (u *PatchK8sObjects) Init(c types.Common, properties cue.Value) error {
	var err error

	u.logger = logrus.WithField("action", typeName)

	u.prop = Properties{}

	// Parse properties.
	err = u.prop.parse(properties)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties")
	}
	u.logger.Debugf("parsed propertise: %v", u.prop)

	u.c = c.Client

	u.logger.Debugf("initialized")

	return nil
}

func (u *PatchK8sObjects) Type() string {
	return typeName
}

func (u *PatchK8sObjects) New() types.Action {
	return &PatchK8sObjects{}
}

func (u *PatchK8sObjects) AllowConcurrency() bool {
	return u.prop.AllowConcurrency
}

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//go:generate ../../../../hack/generate-properties-const-from-cue.sh properties.cue

type Properties struct {
	PatchTarget      PatchTarget `json:"patchTarget"`
	Patch            string      `json:"patch"`
	AllowConcurrency bool        `json:"allowConcurrency"`
}

type PatchTarget struct {
	APIVersion     string            `json:"apiVersion"`
	Kind           string            `json:"kind"`
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	LabelSelectors map[string]string `json:"labelSelectors"`
}

// parse parses, evaluate, validate, and apply defaults.
func (p *Properties) parse(prop cue.Value) error {
	var err error

	// Evaluate and Validate config using properties.cue
	cueCtx := cuecontext.New()

	str, err := utilcue.Marshal(prop)
	if err != nil {
		return err
	}

	v := cueCtx.CompileString(propertiesCUETemplate + str)

	b, err := v.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		return err
	}

	return nil
}
