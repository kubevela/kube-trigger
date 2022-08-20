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

func (r *Reconciler) createServiceAccount(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	update bool,
) error {
	sa := template.GetServiceAccount()

	sa.Name = kt.Name
	sa.Namespace = kt.Namespace

	utils.SetOwnerReference(sa, *kt)

	var err error
	if update {
		logger.Infof("updating ServiceAccount: %s", types.NamespacedName{
			Namespace: sa.Namespace,
			Name:      sa.Name,
		}.String())
		err = r.Update(ctx, sa)
	} else {
		logger.Infof("creating new ServiceAccount: %s", types.NamespacedName{
			Namespace: sa.Namespace,
			Name:      sa.Name,
		}.String())
		err = r.Create(ctx, sa)
	}
	if err != nil {
		return err
	}

	utils.UpdateResource(kt, standardv1alpha1.Resource{
		APIVersion: v1.SchemeGroupVersion.String(),
		Kind:       reflect.TypeOf(v1.ServiceAccount{}).Name(),
		Name:       sa.Name,
		Namespace:  sa.Namespace,
	})

	return nil
}

func (r *Reconciler) deleteServiceAccount(ctx context.Context, namespacedName types.NamespacedName) error {
	sa := template.GetServiceAccount()

	sa.Name = namespacedName.Name
	sa.Namespace = namespacedName.Namespace

	logger.Infof("deleting existing ServiceAccount: %s", namespacedName.String())
	return client.IgnoreNotFound(r.Delete(ctx, sa))
}

func (r *Reconciler) ReconcileServiceAccount(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	req ctrl.Request,
) error {
	if kt.GetUID() == "" {
		return r.deleteServiceAccount(ctx, req.NamespacedName)
	}

	var err error

	sa := v1.ServiceAccount{}
	err = r.Get(ctx, utils.GetNamespacedName(kt), &sa)

	if err == nil {
		return r.createServiceAccount(ctx, kt, true)
	}
	if apierrors.IsNotFound(err) {
		return r.createServiceAccount(ctx, kt, false)
	}

	return err
}
