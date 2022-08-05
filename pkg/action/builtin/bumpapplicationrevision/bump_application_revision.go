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

package bumpapplicationrevision

import (
	"context"
	"strconv"

	"cuelang.org/go/cue"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/action/utils"
	"github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"github.com/oam-dev/kubevela/pkg/oam"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	typeName              = "bump-application-revision"
	initialPublishVersion = "1"
)

// BumpApplicationRevision bumps Application Revision. Use case:
// https://github.com/kubevela/kubevela/issues/4418
type BumpApplicationRevision struct {
	c      client.Client
	prop   Properties
	logger *logrus.Entry
}

var _ types.Action = &BumpApplicationRevision{}

func (bar *BumpApplicationRevision) Run(ctx context.Context, sourceType string, event interface{}, data interface{}) error {
	var err error

	bar.logger.Infof("running, event souce: %s", sourceType)

	appList := v1beta1.ApplicationList{}
	var listOptions []client.ListOption
	if bar.prop.Namespace != "" {
		listOptions = append(listOptions, client.InNamespace(bar.prop.Namespace))
	}
	if len(bar.prop.LabelSelectors) > 0 {
		selector := client.MatchingLabels{}
		for k, v := range bar.prop.LabelSelectors {
			selector[k] = v
		}
		listOptions = append(listOptions, selector)
	}

	err = bar.c.List(ctx, &appList, listOptions...)
	if err != nil {
		bar.logger.Errorf("cannot list %v", bar.prop)
		return err
	}

	var appSlice []v1beta1.Application

	for _, app := range appList.Items {
		// Do name filtering.
		targetName := bar.prop.Name
		if targetName != "" && app.GetName() != targetName {
			continue
		}
		appSlice = append(appSlice, app)
	}

	bar.logger.Infof("found %d apps to bump", len(appSlice))

	// Bump each app
	for i, app := range appSlice {
		err = bar.bumpApp(ctx, &app)
		if err != nil {
			return err
		}
		bar.logger.Infof("%d: app %s bumped", i+1, app.GetName())
	}

	return nil
}

func (bar *BumpApplicationRevision) bumpApp(ctx context.Context, app *v1beta1.Application) error {
	annotations := app.GetAnnotations()

	// If no app.oam.dev/publishVersion already, add it.
	if _, ok := annotations[oam.AnnotationPublishVersion]; !ok {
		annotations[oam.AnnotationPublishVersion] = initialPublishVersion
	}

	// Bump app.oam.dev/publishVersion already
	previous := annotations[oam.AnnotationPublishVersion]
	intVal, err := strconv.ParseInt(previous, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "error when parsing AnnotationPublishVersion")
	}
	bumpedIntVal := intVal + 1
	annotations[oam.AnnotationPublishVersion] = strconv.FormatInt(bumpedIntVal, 10)

	// Update app using new rev.
	bar.logger.Infof("bumping apprev %s from %d to %d", app.GetName(), intVal, bumpedIntVal)
	return bar.c.Update(ctx, app)
}

func (bar *BumpApplicationRevision) Init(c types.Common, properties cue.Value) error {
	var err error
	bar.logger = logrus.WithField("action", typeName)
	bar.prop = Properties{}
	err = bar.prop.parse(properties)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties")
	}
	bar.logger.Debugf("parsed propertise: %v", bar.prop)
	bar.c = c.Client
	bar.logger.Debugf("initialized")
	return nil
}

func (bar *BumpApplicationRevision) Type() string {
	return typeName
}

func (bar *BumpApplicationRevision) New() types.Action {
	return &BumpApplicationRevision{}
}

func (bar *BumpApplicationRevision) AllowConcurrency() bool {
	return false
}

// This will make properties.cue into our go code. We will use it to validate user-provided config.
//go:generate ../../../../hack/generate-properties-const-from-cue.sh properties.cue

type Properties struct {
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	LabelSelectors map[string]string `json:"labelSelectors"`
}

func (p *Properties) parse(prop cue.Value) error {
	return utils.ValidateAndUnMarshal(propertiesCUETemplate, prop, p)
}
