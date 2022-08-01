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

package client

import (
	"github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var k8sClient *client.Client

// GetClient gets a client. It creates a one if not already created. Subsequent
// call will return the previously created one.
func GetClient() (*client.Client, error) {
	if k8sClient != nil {
		return k8sClient, nil
	}

	conf := ctrl.GetConfigOrDie()
	err := v1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	c, err := client.New(conf, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	k8sClient = &c
	return k8sClient, nil
}
