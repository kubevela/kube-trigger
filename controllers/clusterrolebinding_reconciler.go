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

package controllers

import (
	"context"
	"reflect"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/template"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KubeTriggerReconciler) createClusterRoleBinding(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	update bool,
) error {
	crb := template.GetClusterRoleBinding()

	// TODO(charlie0129): allow user to set custom privileges instead of cluster-admin.

	crb.Name = kt.Name
	crb.Namespace = kt.Namespace
	// It must have one subject.
	crb.Subjects[0].Name = kt.Name
	crb.Subjects[0].Namespace = kt.Namespace

	var err error
	if update {
		logger.Infof("updating ClusterRoleBinding: %s", types.NamespacedName{
			Namespace: crb.Namespace,
			Name:      crb.Name,
		}.String())
		err = r.Update(ctx, crb)
	} else {
		logger.Infof("creating new ClusterRoleBinding: %s", types.NamespacedName{
			Namespace: crb.Namespace,
			Name:      crb.Name,
		}.String())
		err = r.Create(ctx, crb)
	}
	if err != nil {
		return err
	}

	updateResource(kt, standardv1alpha1.Resource{
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Kind:       reflect.TypeOf(rbacv1.ClusterRoleBinding{}).Name(),
		Name:       crb.Name,
		Namespace:  crb.Namespace,
	})

	return nil
}

func (r *KubeTriggerReconciler) deleteClusterRoleBinding(
	ctx context.Context,
	namespacedName types.NamespacedName,
) error {
	crb := template.GetClusterRoleBinding()

	crb.Name = namespacedName.Name
	crb.Namespace = namespacedName.Namespace

	logger.Infof("deleting existing ClusterRoleBinding: %s", namespacedName.String())
	return client.IgnoreNotFound(r.Delete(ctx, crb))
}

func (r *KubeTriggerReconciler) ReconcileClusterRoleBinding(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	req ctrl.Request,
) error {
	if kt.GetUID() == "" {
		return r.deleteClusterRoleBinding(ctx, req.NamespacedName)
	}

	var err error

	crb := rbacv1.ClusterRoleBinding{}
	err = r.Get(ctx, getNamespacedName(*kt), &crb)

	if err == nil {
		return r.createClusterRoleBinding(ctx, kt, true)
	}

	if errors.IsNotFound(err) {
		return r.createClusterRoleBinding(ctx, kt, false)
	}

	return err
}
