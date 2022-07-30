package updatek8sobjects

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/imdario/mergo"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	krwevent "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/event"
	utilcue "github.com/kubevela/kube-trigger/pkg/utils/cue"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	patchFieldName            = "patch"
	patchTargetFieldName      = "patchTarget"
	allowConcurrencyFieldName = "allowConcurrency"
	outputFieldName           = "output"
)
const (
	typeName = "update-k8s-objects"
)

type Properties struct {
	PatchTarget      PatchTarget
	Patch            string
	AllowConcurrency bool
}

type PatchTarget struct {
	APIVersion     string            `json:"apiVersion"`
	Kind           string            `json:"kind"`
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	LabelSelectors map[string]string `json:"labelSelectors"`
}

// TODO: use cue to validate
func (p *Properties) parse(v cue.Value) error {
	var err error

	vPatch := v.LookupPath(cue.ParsePath(patchFieldName))
	if vPatch.Err() != nil {
		return vPatch.Err()
	}

	p.Patch, err = vPatch.String()
	if err != nil {
		return err
	}

	pt := v.LookupPath(cue.ParsePath(patchTargetFieldName))
	if pt.Err() != nil {
		return pt.Err()
	}

	str, err := pt.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(str), &p.PatchTarget)
	if err != nil {
		return err
	}

	vAllowConcurrent := v.LookupPath(cue.ParsePath(allowConcurrencyFieldName))
	p.AllowConcurrency, err = vAllowConcurrent.Bool()
	if err != nil {
		p.AllowConcurrency = false
	}

	return nil
}

type UpdateK8sObject struct {
	c      client.Client
	prop   Properties
	logger *logrus.Entry
}

func (u *UpdateK8sObject) Run(ctx context.Context, sourceType string, event interface{}) error {
	var contextStr string
	u.logger.Debugf("running with event %#v", event)

	e, ok := event.(krwevent.Event)
	if ok {
		jsonByte, err := json.Marshal(e.Obj)
		if err == nil {
			u.logger.Debugf("added context.sourceObject: %s", string(jsonByte))
			contextStr += fmt.Sprintf("context:{sourceObject:%s}\n", string(jsonByte))
		}
	} else {
		u.logger.Debugf("event is not a krw event")
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
		return err
	}

	var objList []unstructured.Unstructured

	for _, un := range unstructuredObjList.Items {
		targetName := u.prop.PatchTarget.Name
		if targetName != "" && un.GetName() != targetName {
			continue
		}
		objList = append(objList, un)
	}

	for _, un := range objList {
		err = u.updateObject(ctx, contextStr, un)
		if err != nil {
			return err
		}
	}

	u.logger.Debugf("finished successfully")

	return nil
}

func (u *UpdateK8sObject) updateObject(ctx context.Context, contextStr string, un unstructured.Unstructured) error {
	jsonByte, err := json.Marshal(un.Object)
	if err == nil {
		u.logger.Debugf("added context.patchTarget: %s", string(jsonByte))
		contextStr += fmt.Sprintf("context:{patchTarget:%s}\n", string(jsonByte))
	}

	cueCtx := cuecontext.New()
	// context+patch string
	v := cueCtx.CompileString(u.prop.Patch + "\n" + contextStr)
	if v.Err() != nil {
		return v.Err()
	}
	vPatch := v.LookupPath(cue.ParsePath(outputFieldName))

	patchOut := make(map[string]interface{})

	err = utilcue.UnMarshal(vPatch, patchOut)
	if err != nil {
		return err
	}

	err = mergo.Merge(&un.Object, patchOut, mergo.WithOverride)
	if err != nil {
		return err
	}

	u.logger.Debugf("merged with patch, ready to update: %v", un.Object)

	err = u.c.Update(ctx, &un)
	if err != nil {
		return err
	}

	return nil
}

func (u *UpdateK8sObject) Init(c types.Common, properties cue.Value) error {
	var err error

	u.prop = Properties{}

	err = u.prop.parse(properties)
	if err != nil {
		return err
	}

	u.c = c.Client

	u.logger = logrus.WithField("action", typeName)
	u.logger.Debugf("initialized")

	return nil
}

func (u *UpdateK8sObject) Type() string {
	return typeName
}

func (u *UpdateK8sObject) New() types.Action {
	return &UpdateK8sObject{}
}

func (u *UpdateK8sObject) AllowConcurrent() bool {
	return u.prop.AllowConcurrency
}
