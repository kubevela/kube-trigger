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

package util

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func IgnoreAlreadyExists(err error) error {
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func IgnoreNotFound(err error) error {
	return client.IgnoreNotFound(err)
}

// ReconcileRetry will reconcile with retry
func ReconcileRetry(r reconcile.Reconciler, req reconcile.Request) {
	Eventually(func() error {
		if _, err := r.Reconcile(context.TODO(), req); err != nil {
			return err
		}
		return nil
	}, 15*time.Second, time.Second).Should(BeNil())
}

// ReconcileRetryAndExpectErr will reconcile and get error
func ReconcileRetryAndExpectErr(r reconcile.Reconciler, req reconcile.Request) {
	Eventually(func() error {
		if _, err := r.Reconcile(context.TODO(), req); err != nil {
			return err
		}
		return nil
	}, 3*time.Second, time.Second).ShouldNot(BeNil())
}

// ReconcileOnce will just reconcile once
func ReconcileOnce(r reconcile.Reconciler, req reconcile.Request) {
	if _, err := r.Reconcile(context.TODO(), req); err != nil {
		fmt.Println(err.Error())
	}
}

// ReconcileOnceAfterFinalizer will reconcile for finalizer
func ReconcileOnceAfterFinalizer(r reconcile.Reconciler, req reconcile.Request) (reconcile.Result, error) {
	// 1st and 2nd time reconcile to add finalizer
	if result, err := r.Reconcile(context.TODO(), req); err != nil {
		return result, err
	}
	if result, err := r.Reconcile(context.TODO(), req); err != nil {
		return result, err
	}
	if result, err := r.Reconcile(context.TODO(), req); err != nil {
		return result, err
	}
	if result, err := r.Reconcile(context.TODO(), req); err != nil {
		return result, err
	}

	return r.Reconcile(context.TODO(), req)
}
