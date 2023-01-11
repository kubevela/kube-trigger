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

package triggerinstance

import (
	"context"
	"fmt"
	"reflect"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/template"
	"github.com/kubevela/kube-trigger/controllers/utils"
	"github.com/kubevela/kube-trigger/pkg/cmd"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	createdByLabel                = "app.kubernetes.io/created-by"
	createdByControllerLabelValue = "kube-trigger-manager"
	// NameLabel is to store name of the crd.
	NameLabel = "app.kubernetes.io/name"
)

func (r *Reconciler) createDeployment(
	ctx context.Context,
	ki *standardv1alpha1.TriggerInstance,
	update bool,
) error {
	deployment := template.GetDeployment()

	deployment.Name = ki.Name
	deployment.Namespace = ki.Namespace
	deployment.Labels[createdByLabel] = createdByControllerLabelValue
	deployment.Labels[NameLabel] = ki.Name
	deployment.Spec.Selector.MatchLabels[NameLabel] = ki.Name
	deployment.Spec.Template.ObjectMeta.Labels[NameLabel] = ki.Name
	deployment.Spec.Template.Spec.Containers[0].Args = workerConfigToArgs(
		deployment.Spec.Template.Spec.Containers[0].Args,
		ki.Spec.WorkerConfig,
	)
	// TODO: use a deterministic version of image or let the use specify it
	// deployment.Spec.Template.Spec.Containers[0].Image = ""
	deployment.Spec.Template.Spec.ServiceAccountName = ki.Name
	deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name = ki.Name

	utils.SetOwnerReference(deployment, ki)

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

	return nil
}

func workerConfigToArgs(args []string, wc *standardv1alpha1.WorkerConfig) []string {
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
	if wc.Timeout != nil {
		args = append(args, flagToArg(cmd.FlagTimeout, wc.Timeout))
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

func (r *Reconciler) reconcileDeployment(ctx context.Context, ki *standardv1alpha1.TriggerInstance) error {
	deployment := appsv1.Deployment{}
	err := r.Get(ctx, utils.GetNamespacedName(ki), &deployment)

	if err == nil {
		return r.createDeployment(ctx, ki, true)
	}
	if apierrors.IsNotFound(err) {
		return r.createDeployment(ctx, ki, false)
	}

	return err
}
