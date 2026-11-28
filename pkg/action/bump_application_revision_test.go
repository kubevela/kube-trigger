/*
Copyright 2026 The KubeVela Authors.

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

package action

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/kubevela/pkg/cue/cuex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const bumpDefinition = "../../config/definition/bump-application-revision.yaml"

// application is an Application as the cluster would return it. An empty
// version leaves the publishVersion annotation out.
func application(name, version string) map[string]interface{} {
	metadata := map[string]interface{}{"name": name, "namespace": "shop"}
	if version != "" {
		metadata["annotations"] = map[string]interface{}{"app.oam.dev/publishVersion": version}
	}
	return map[string]interface{}{"apiVersion": "core.oam.dev/v1beta1", "kind": "Application", "metadata": metadata}
}

// renderBump compiles the action for a Deployment update without calling the
// cluster. found stands in for what its kube.#Get or kube.#List would return.
func renderBump(t *testing.T, properties map[string]interface{}, found ...map[string]interface{}) cue.Value {
	cuex.EnableExternalPackageForDefaultCompiler = false
	b, err := os.ReadFile(bumpDefinition)
	require.NoError(t, err)
	var def struct {
		Spec struct {
			Templates map[string]string `json:"templates"`
		} `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(b, &def))

	opts := []cuex.CompileOption{
		cuex.WithExtraData("parameter", properties),
		cuex.WithExtraData("context", deploymentEvent()),
		cuex.DisableResolveProviderFunctions{},
	}
	if selector, ok := properties["nameSelector"].(map[string]interface{}); ok && selector["matchingLabels"] != nil {
		items := make([]interface{}, 0, len(found))
		for _, app := range found {
			items = append(items, app)
		}
		opts = append(opts, cuex.WithExtraData("list", map[string]interface{}{"$returns": map[string]interface{}{"items": items}}))
	} else if len(found) == 1 {
		opts = append(opts, cuex.WithExtraData("get", map[string]interface{}{"$returns": found[0]}))
	}
	v, err := cuex.DefaultCompiler.Get().CompileStringWithOptions(context.Background(), def.Spec.Templates["main.cue"], opts...)
	require.NoError(t, err)
	require.NoError(t, v.Err())
	return v
}

// bumps returns, for every patch the action would make, the Application's name
// and the publishVersion it would be given.
func bumps(t *testing.T, v cue.Value) map[string]string {
	out := map[string]string{}
	fields, err := v.Fields()
	require.NoError(t, err)
	for fields.Next() {
		if !strings.HasPrefix(fields.Selector().String(), `"patch-`) {
			continue
		}
		params := fields.Value().LookupPath(cue.ParsePath("$params"))
		name, err := params.LookupPath(cue.ParsePath("resource.metadata.name")).String()
		require.NoError(t, err)
		version, err := params.LookupPath(cue.ParsePath(`patch.data.metadata.annotations."app.oam.dev/publishVersion"`)).String()
		require.NoError(t, err)
		out[name] = version
	}
	return out
}

func str(t *testing.T, v cue.Value, path string) string {
	s, err := v.LookupPath(cue.ParsePath(path)).String()
	require.NoError(t, err, path)
	return s
}

func TestBumpTheApplicationNamedAfterTheSourceObject(t *testing.T) {
	v := renderBump(t, map[string]interface{}{"namespace": "shop"}, application("web", "5"))

	assert.Equal(t, "web", str(t, v, "get.$params.resource.metadata.name"))
	assert.Equal(t, map[string]string{"web": "6"}, bumps(t, v))
}

func TestBumpTheApplicationNamedByALabel(t *testing.T) {
	v := renderBump(t, map[string]interface{}{
		"namespace":    "shop",
		"nameSelector": map[string]interface{}{"fromLabel": "app"},
	}, application("web", "1"))

	// deploymentEvent's Deployment carries the label app=web.
	assert.Equal(t, "web", str(t, v, "get.$params.resource.metadata.name"))
	assert.Equal(t, map[string]string{"web": "2"}, bumps(t, v))
}

func TestBumpEveryApplicationMatchingLabels(t *testing.T) {
	v := renderBump(t, map[string]interface{}{
		"namespace":    "shop",
		"nameSelector": map[string]interface{}{"matchingLabels": map[string]interface{}{"team": "blue"}},
	}, application("a", "7"), application("b", "not-a-number"), application("c", ""), application("d", "41"))

	// Only Applications whose publishVersion is a whole number are bumped.
	assert.Equal(t, map[string]string{"a": "8", "d": "42"}, bumps(t, v))
}

func TestBumpMatchingNoApplication(t *testing.T) {
	v := renderBump(t, map[string]interface{}{
		"namespace":    "shop",
		"nameSelector": map[string]interface{}{"matchingLabels": map[string]interface{}{"team": "blue"}},
	})

	assert.Empty(t, bumps(t, v))
}

// With no namespace given, the Applications are looked for in the source
// object's namespace, as one concrete value the provider can use.
func TestBumpNamespaceDefaultsToTheSourceObjects(t *testing.T) {
	testcases := map[string]struct {
		properties map[string]interface{}
		path       string
	}{
		"by name": {
			properties: map[string]interface{}{},
			path:       "get.$params.resource.metadata.namespace",
		},
		"by a label": {
			properties: map[string]interface{}{"nameSelector": map[string]interface{}{"fromLabel": "app"}},
			path:       "get.$params.resource.metadata.namespace",
		},
		"by matching labels": {
			properties: map[string]interface{}{"nameSelector": map[string]interface{}{"matchingLabels": map[string]interface{}{"team": "blue"}}},
			path:       "list.$params.filter.namespace",
		},
	}
	names := make([]string, 0, len(testcases))
	for name := range testcases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		tc := testcases[name]
		t.Run(name, func(t *testing.T) {
			v := renderBump(t, tc.properties)
			namespace := v.LookupPath(cue.ParsePath(tc.path))
			require.NoError(t, namespace.Validate(cue.Concrete(true)), "namespace must be one concrete value")
			got, err := namespace.String()
			require.NoError(t, err)
			assert.Equal(t, "shop", got)
		})
	}
}
