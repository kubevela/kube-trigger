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

package bumpapplicationrevision_test

import (
	"context"
	"strings"
	"testing"

	bar "github.com/kubevela/kube-trigger/pkg/action/builtin/bumpapplicationrevision"
	"github.com/kubevela/kube-trigger/pkg/util"
	"github.com/oam-dev/kubevela-core-api/apis/core.oam.dev/v1beta1"
	"github.com/oam-dev/kubevela-core-api/pkg/oam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestParseProperties(t *testing.T) {
	cases := map[string]struct {
		prop     map[string]interface{}
		expected bar.Properties
		err      bool
	}{
		"valid empty prop": {
			prop:     map[string]interface{}{},
			expected: bar.Properties{},
			err:      false,
		},
		"valid prop with labelSelectors": {
			prop: map[string]interface{}{
				"labelSelectors": map[string]string{
					"app": "none",
				},
			},
			expected: bar.Properties{
				LabelSelectors: map[string]string{
					"app": "none",
				},
			},
			err: false,
		},
		"valid prop with all fields": {
			prop: map[string]interface{}{
				"name":      "name",
				"namespace": "namespace",
				"labelSelectors": map[string]string{
					"app": "none",
				},
				"random-field": "",
			},
			expected: bar.Properties{
				Name:      "name",
				Namespace: "namespace",
				LabelSelectors: map[string]string{
					"app": "none",
				},
			},
			err: false,
		},
		"invalid prop type": {
			prop: map[string]interface{}{
				"name":      1,
				"namespace": struct{ A string }{},
				"labelSelectors": map[string]int{
					"app": 3,
				},
				"random-field": "",
			},
			expected: bar.Properties{},
			err:      true,
		},
	}

	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			p := bar.Properties{}
			err := p.Parse(v.prop)
			if v.err {
				r.Error(err)
			} else {
				r.NoError(err)
				r.Equal(p, v.expected)
			}
		})
	}
}

var _ = Describe("Test BumpApplicationRevision", Ordered, func() {
	ctx := context.TODO()

	BeforeEach(func() {
		By("Add test Applications")
		err := k8sClient.Create(ctx, ns.DeepCopy())
		Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
		for _, a := range apps {
			err := k8sClient.Create(ctx, parseApp(a))
			Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
		}
	})

	AfterAll(func() {
		By("Clean up test applications")
		for _, a := range apps {
			err := k8sClient.Delete(ctx, parseApp(a))
			Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
		}
		err := k8sClient.Delete(ctx, ns.DeepCopy())
		Expect(util.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
	})

	It("Bump app with no apprev annotation", func() {
		b := bar.BumpApplicationRevision{}
		err := b.Init(actionCommon, map[string]interface{}{
			"name": "no-annotation",
		})
		Expect(err).NotTo(HaveOccurred())

		err = b.Run(ctx, "", nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		app := &v1beta1.Application{}
		err = k8sClient.Get(ctx, apitypes.NamespacedName{
			Namespace: "default",
			Name:      "no-annotation",
		}, app)
		Expect(err).NotTo(HaveOccurred())

		Expect(app.Annotations[oam.AnnotationPublishVersion]).To(Equal("2"))

		// App in another ns should be updated as well
		err = k8sClient.Get(ctx, apitypes.NamespacedName{
			Namespace: "multiple-apps",
			Name:      "no-annotation",
		}, app)
		Expect(err).NotTo(HaveOccurred())

		Expect(app.Annotations[oam.AnnotationPublishVersion]).To(Equal("2"))
	})

	It("Bump all apps in a namespace", func() {
		b := bar.BumpApplicationRevision{}
		err := b.Init(actionCommon, map[string]interface{}{
			"namespace": "multiple-apps",
		})
		Expect(err).NotTo(HaveOccurred())

		err = b.Run(ctx, "", nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		apps := v1beta1.ApplicationList{}
		err = k8sClient.List(ctx, &apps, client.InNamespace("multiple-apps"))
		Expect(err).NotTo(HaveOccurred())

		for _, app := range apps.Items {
			if strings.HasPrefix(app.Name, "multi-") {
				Expect(app.Annotations[oam.AnnotationPublishVersion]).To(Equal("2"))
			}
		}
	})

	It("Bump apps using selectors", func() {
		b := bar.BumpApplicationRevision{}
		err := b.Init(actionCommon, map[string]interface{}{
			"labelSelectors": map[string]string{
				"my-label": "my-value",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		err = b.Run(ctx, "", nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		apps := v1beta1.ApplicationList{}
		err = k8sClient.List(ctx, &apps)
		Expect(err).NotTo(HaveOccurred())

		for _, app := range apps.Items {
			if strings.HasPrefix(app.Name, "this-will-be") {
				Expect(app.Annotations[oam.AnnotationPublishVersion]).To(Equal("11"))
			}
			if strings.HasPrefix(app.Name, "this-will-not-be") {
				Expect(app.Annotations[oam.AnnotationPublishVersion]).To(Equal("10"))
			}
		}
	})
})

func parseApp(str string) *v1beta1.Application {
	app := &v1beta1.Application{}
	err := yaml.Unmarshal([]byte(str), app)
	Expect(err).NotTo(HaveOccurred())
	return app
}

var (
	apps = []string{
		// Bump app with no apprev annotation
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: no-annotation
  namespace: default
spec:
  components: [ ]
`,
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: no-annotation
  namespace: multiple-apps
spec:
  components: [ ]
`,
		// Bump all apps in a namespace
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: multi-no-annotation
  namespace: multiple-apps
spec:
  components: [ ]
`,
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  annotations:
    app.oam.dev/publishVersion: "1"
  name: multi-have-annotation
  namespace: multiple-apps
spec:
  components: [ ]
`,
		// Bump apps using selectors
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  annotations:
    app.oam.dev/publishVersion: "10"
  name: this-will-be-updated-2
  labels:
    my-label: my-value
  namespace: default
spec:
  components: [ ]
`,
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  annotations:
    app.oam.dev/publishVersion: "10"
  name: this-will-be-updated-1
  labels:
    my-label: my-value
  namespace: default
spec:
  components: [ ]
`,
		`
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  annotations:
    app.oam.dev/publishVersion: "10"
  name: this-will-not-be-updated
  namespace: default
spec:
  components: [ ]
`,
	}
	ns = v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "multiple-apps"}}
)
