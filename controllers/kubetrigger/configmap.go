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

package kubetrigger

import (
	"context"
	"reflect"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/template"
	"github.com/kubevela/kube-trigger/controllers/utils"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) createConfigMap(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
) error {
	cm := template.GetConfigMap()
	cm.Name = kt.Name
	cm.Namespace = kt.Namespace

	utils.SetOwnerReference(cm, *kt)

	var err error
	logger.Infof("creating new ConfigMap: %s", types.NamespacedName{
		Namespace: cm.Namespace,
		Name:      cm.Name,
	}.String())
	err = r.Create(ctx, cm)
	if err != nil {
		return err
	}

	utils.UpdateResource(kt, standardv1alpha1.Resource{
		APIVersion: v1.SchemeGroupVersion.String(),
		Kind:       reflect.TypeOf(v1.ConfigMap{}).Name(),
		Name:       cm.Name,
		Namespace:  cm.Namespace,
	})

	return nil
}

func (r *Reconciler) deleteConfigMap(ctx context.Context, namespacedName types.NamespacedName) error {
	cm := template.GetConfigMap()

	cm.Name = namespacedName.Name
	cm.Namespace = namespacedName.Namespace

	logger.Infof("deleting existing ConfigMap: %s", namespacedName.String())
	return client.IgnoreNotFound(r.Delete(ctx, cm))
}

func (r *Reconciler) ReconcileConfigMap(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	req ctrl.Request,
) error {
	if kt.GetUID() == "" {
		return r.deleteConfigMap(ctx, req.NamespacedName)
	}

	var err error

	cm := v1.ConfigMap{}
	err = r.Get(ctx, utils.GetNamespacedName(kt), &cm)

	if apierrors.IsNotFound(err) {
		return r.createConfigMap(ctx, kt)
	}

	return err
}
