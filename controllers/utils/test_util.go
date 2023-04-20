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
	"os/exec"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
