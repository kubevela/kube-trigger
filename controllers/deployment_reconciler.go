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
	"fmt"
	"reflect"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/template"
	"github.com/kubevela/kube-trigger/pkg/cmd"
	"github.com/kubevela/kube-trigger/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CreatedByLabel                = "app.kubernetes.io/created-by"
	CreatedByControllerLabelValue = "kube-trigger-manager"

	ComponentLabel      = "app.kubernetes.io/component"
	ComponentLabelValue = "kube-trigger"

	// VersionLabel is controller version that created this pod.
	VersionLabel = "app.kubernetes.io/version"

	// NameLabel is to store name of the crd.
	NameLabel = "app.kubernetes.io/name"
)

func (r *KubeTriggerReconciler) createDeployment(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	update bool,
) error {
	deployment := template.GetDeployment()

	deployment.Name = kt.Name
	deployment.Namespace = kt.Namespace
	deployment.Labels[CreatedByLabel] = CreatedByControllerLabelValue
	deployment.Labels[ComponentLabel] = ComponentLabelValue
	deployment.Labels[VersionLabel] = version.Version
	deployment.Labels[NameLabel] = kt.Name
	deployment.Spec.Selector.MatchLabels[NameLabel] = kt.Name
	deployment.Spec.Template.ObjectMeta.Labels[NameLabel] = kt.Name
	deployment.Spec.Template.Spec.Containers[0].Args = workerConfigToArgs(kt.Spec.WorkerConfig)
	deployment.Spec.Template.Spec.ServiceAccountName = kt.Name
	deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name = kt.Name

	setOwnerReference(deployment, *kt)

	var err error
	if update {
		logger.Infof("updating Deployment: %s", types.NamespacedName{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
		}.String())
		err = r.Update(ctx, deployment)
	} else {
		logger.Infof("creating new Deployment: %s", types.NamespacedName{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
		}.String())
		err = r.Create(ctx, deployment)
	}
	if err != nil {
		return err
	}

	updateResource(kt, standardv1alpha1.Resource{
		APIVersion: appsv1.SchemeGroupVersion.String(),
		Kind:       reflect.TypeOf(appsv1.Deployment{}).Name(),
		Name:       deployment.Name,
		Namespace:  deployment.Namespace,
	})

	return nil
}

func workerConfigToArgs(wc *standardv1alpha1.WorkerConfig) []string {
	args := []string{}
	if wc == nil {
		return args
	}
	if wc.MaxRetry != nil {
		args = append(args, flagToArg(cmd.FlagMaxRetry, wc.MaxRetry))
	}
	if wc.RetryDelay != nil {
		args = append(args, flagToArg(cmd.FlagRetryDelay, wc.RetryDelay))
	}
	if wc.PerWorkerQPS != nil {
		args = append(args, flagToArg(cmd.FlagPerWorkerQPS, wc.PerWorkerQPS))
	}
	if wc.QueueSize != nil {
		args = append(args, flagToArg(cmd.FlagQueueSize, wc.QueueSize))
	}
	if wc.JobTimeout != nil {
		args = append(args, flagToArg(cmd.FlagTimeout, wc.JobTimeout))
	}
	if wc.WorkerCount != nil {
		args = append(args, flagToArg(cmd.FlagWorkers, wc.WorkerCount))
	}
	return args
}

func flagToArg(flag string, value interface{}) string {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		return fmt.Sprintf("--%s=%v", flag, v.Elem())
	}
	return fmt.Sprintf("--%s=%v", flag, value)
}

func (r *KubeTriggerReconciler) deleteDeployment(ctx context.Context, namespacedName types.NamespacedName) error {
	deployment := template.GetDeployment()

	deployment.Name = namespacedName.Name
	deployment.Namespace = namespacedName.Namespace

	logger.Infof("deleting existing Deployment: %s", namespacedName.String())
	return client.IgnoreNotFound(r.Delete(ctx, deployment))
}

func (r *KubeTriggerReconciler) ReconcileDeployment(
	ctx context.Context,
	kt *standardv1alpha1.KubeTrigger,
	req ctrl.Request,
) error {
	if kt.GetUID() == "" {
		return r.deleteDeployment(ctx, req.NamespacedName)
	}

	var err error

	deployment := appsv1.Deployment{}
	err = r.Get(ctx, getNamespacedName(*kt), &deployment)

	if err == nil {
		return r.createDeployment(ctx, kt, true)
	}
	if apierrors.IsNotFound(err) {
		return r.createDeployment(ctx, kt, false)
	}

	return err
}
