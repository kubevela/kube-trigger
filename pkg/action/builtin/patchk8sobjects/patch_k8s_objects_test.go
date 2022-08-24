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

package patchk8sobjects_test

import (
	"context"

	pko "github.com/kubevela/kube-trigger/pkg/action/builtin/patchk8sobjects"
	"github.com/kubevela/kube-trigger/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Test context in patch", func() {
	ctx := context.TODO()
	var props map[string]interface{}

	BeforeEach(func() {
		props = map[string]interface{}{
			"patchTarget": map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"namespace":  "default",
			},
		}

		By("create test configmaps")
		cm := v1.ConfigMap{}
		cm.Namespace = "default"
		cm.Name = "test-cm"
		cm.Data = map[string]string{"data": ""}
		err := k8sClient.Create(ctx, &cm)
		Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
	})

	test := func(patch string, event, data interface{}, expected string) func() {
		return func() {
			p := pko.PatchK8sObjects{}
			props["patch"] = patch
			err := p.Init(actionCommon, props)
			Expect(err).NotTo(HaveOccurred())

			err = p.Run(ctx, "", event, data, nil)
			Expect(err).NotTo(HaveOccurred())

			cm := v1.ConfigMap{}
			err = k8sClient.Get(ctx, apitypes.NamespacedName{
				Namespace: "default",
				Name:      "test-cm",
			}, &cm)
			Expect(err).NotTo(HaveOccurred())

			Expect(cm.Data["data"]).To(Equal(expected))
		}
	}

	It("no context", test(
		`output: data: data: "no context"`,
		nil,
		nil,
		"no context",
	))

	It("use context.event", test(
		`output: data: data: context.event`,
		"use context.event",
		nil,
		"use context.event",
	))

	It("use context.data", test(
		`output: data: data: context.data`,
		nil,
		"use context.data",
		"use context.data",
	))

	It("use context.target", test(
		`output: data: data: context.target.kind`,
		nil,
		nil,
		"ConfigMap",
	))
})

var _ = Describe("General tests", func() {
	ctx := context.TODO()
	var props map[string]interface{}

	It("invalid prop", func() {
		p := pko.PatchK8sObjects{}
		props = make(map[string]interface{})
		err := p.Init(actionCommon, props)
		Expect(err).To(HaveOccurred())
	})

	It("invalid patch", func() {
		cm := v1.ConfigMap{}
		cm.Namespace = "default"
		cm.Name = "test-cm"
		cm.Data = map[string]string{"data": ""}
		err := k8sClient.Create(ctx, &cm)
		Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())

		p := pko.PatchK8sObjects{}
		props = map[string]interface{}{
			"patchTarget": map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"namespace":  "default",
			},
			"patch": ":,./,",
		}
		err = p.Init(actionCommon, props)
		Expect(err).NotTo(HaveOccurred())

		err = p.Run(ctx, "", nil, nil, nil)
		Expect(err).To(HaveOccurred())
	})

	It("test patchTarget.name and patchTarget.labelSelectors", func() {
		cm1 := v1.ConfigMap{}
		cm1.Namespace = "default"
		cm1.Name = "test-cm-1"
		cm1.Data = map[string]string{"data": ""}
		err := k8sClient.Create(ctx, &cm1)
		Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
		cm2 := v1.ConfigMap{}
		cm2.Namespace = "default"
		cm2.Name = "test-cm-2"
		cm2.Labels = map[string]string{"my-label": "my-value"}
		cm2.Data = map[string]string{"data": ""}
		err = k8sClient.Create(ctx, &cm2)
		Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())

		By("Name restrictions: only test-cm-1 should be patched")
		p := pko.PatchK8sObjects{}
		props = map[string]interface{}{
			"patchTarget": map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"namespace":  "default",
				"name":       "test-cm-1",
			},
			"patch": `output: data: data: "only test-cm-1"`,
		}
		err = p.Init(actionCommon, props)
		Expect(err).NotTo(HaveOccurred())

		err = p.Run(ctx, "", nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		cm := v1.ConfigMap{}
		err = k8sClient.Get(ctx, apitypes.NamespacedName{
			Namespace: "default",
			Name:      "test-cm-1",
		}, &cm)
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Data["data"]).To(Equal("only test-cm-1"))

		err = k8sClient.Get(ctx, apitypes.NamespacedName{
			Namespace: "default",
			Name:      "test-cm-2",
		}, &cm)
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Data["data"]).To(Equal(""))

		By("Label selectors: only test-cm-2 should be patched")
		props = map[string]interface{}{
			"patchTarget": map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"namespace":  "default",
				"labelSelectors": map[string]string{
					"my-label": "my-value",
				},
			},
			"patch": `output: data: data: "only test-cm-2"`,
		}
		err = p.Init(actionCommon, props)
		Expect(err).NotTo(HaveOccurred())

		err = p.Run(ctx, "", nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Get(ctx, apitypes.NamespacedName{
			Namespace: "default",
			Name:      "test-cm-1",
		}, &cm)
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Data["data"]).To(Equal("only test-cm-1"))

		err = k8sClient.Get(ctx, apitypes.NamespacedName{
			Namespace: "default",
			Name:      "test-cm-2",
		}, &cm)
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Data["data"]).To(Equal("only test-cm-2"))
	})
})
