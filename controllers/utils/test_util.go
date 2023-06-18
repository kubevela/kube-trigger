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

package utils

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

// ReconcileOnce will just reconcile once.
func ReconcileOnce(r reconcile.Reconciler, req reconcile.Request) error {
	if _, err := r.Reconcile(context.TODO(), req); err != nil {
		return err
	}
	return nil
}

// InstallDefaultDefinition install the default template
func InstallDefaultDefinition() error {
	defaultDefinitionPath := filepath.Join("..", "..", "config", "definition", "default.yaml")
	cmd := exec.Command("kubectl", "apply", "-f", defaultDefinitionPath)
	return cmd.Run()
}

// UnInstallDefaultDefinition uninstall the default template
func UnInstallDefaultDefinition() error {
	defaultDefinitionPath := filepath.Join("..", "..", "config", "definition", "default.yaml")
	cmd := exec.Command("kubectl", "delete", "-f", defaultDefinitionPath)
	return cmd.Run()
}

// InstallDefinition install definition before test
func InstallDefinition(ctx context.Context, cli client.Client, defPath string) error {
	b, err := os.ReadFile(defPath)
	if err != nil {
		return err
	}
	s := string(b)
	defJSON, err := yaml.YAMLToJSON([]byte(s))
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(defJSON, u); err != nil {
		return err
	}
	return cli.Create(ctx, u.DeepCopy())
}

// AlreadyExistMatcher matches the error to be already exist
type AlreadyExistMatcher struct {
}

// Match matches error.
func (matcher AlreadyExistMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}
	actualError := actual.(error)
	return apierrors.IsAlreadyExists(actualError), nil
}

// FailureMessage builds an error message.
func (matcher AlreadyExistMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be already exist")
}

// NegatedFailureMessage builds an error message.
func (matcher AlreadyExistMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be already exist")
}

var _ types.GomegaMatcher = NotFoundMatcher{}

// NotFoundMatcher matches the error to be not found.
type NotFoundMatcher struct {
}

// Match matches the api error.
func (matcher NotFoundMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}
	actualError := actual.(error)
	return apierrors.IsNotFound(actualError), nil
}

// FailureMessage builds an error message.
func (matcher NotFoundMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be not found")
}

// NegatedFailureMessage builds an error message.
func (matcher NotFoundMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be not found")
}
