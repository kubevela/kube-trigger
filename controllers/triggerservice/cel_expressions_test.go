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

package triggerservice

import (
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
)

func TestCELExpressionsFor(t *testing.T) {
	annotated := func(v string) *standardv1alpha1.TriggerService {
		return &standardv1alpha1.TriggerService{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"trigger.oam.dev/cel-expressions": v}}}
	}
	plain := &standardv1alpha1.TriggerService{}
	on := Reconciler{CELExpressions: true}
	optInOnly := Reconciler{CELExpressions: true, RequireCELExpressionOptIn: true}

	testcases := map[string]struct {
		r    Reconciler
		ts   *standardv1alpha1.TriggerService
		want bool
	}{
		"on, no annotation":               {r: on, ts: plain, want: true},
		"on, opted out":                   {r: on, ts: annotated("false"), want: false},
		"on, other values change nothing": {r: on, ts: annotated("no"), want: true},
		"off for every TriggerService":    {r: Reconciler{}, ts: annotated("true"), want: false},
		"opt-in required, opted in":       {r: optInOnly, ts: annotated("true"), want: true},
		"opt-in required, not opted in":   {r: optInOnly, ts: plain, want: false},
		"opt-in required, other value":    {r: optInOnly, ts: annotated("yes"), want: false},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.r.celExpressionsFor(tc.ts))
		})
	}
}

// workerArgs renders the default worker template's container args.
func workerArgs(t *testing.T, triggerService map[string]string) []string {
	b, err := os.ReadFile("../../config/definition/default.yaml")
	require.NoError(t, err)
	var def struct {
		Spec struct {
			Templates map[string]string `json:"templates"`
		} `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(b, &def))
	v := cuecontext.New().CompileString(def.Spec.Templates["main.cue"])
	v = v.FillPath(cue.ParsePath("triggerService"), triggerService)
	var args []string
	require.NoError(t, v.LookupPath(cue.ParsePath("deployment.spec.template.spec.containers[0].args")).Decode(&args))
	return args
}

func TestWorkerTemplatePassesCELExpressions(t *testing.T) {
	// The worker defaults to on, so the manager's choice is passed both ways.
	for _, v := range []string{"true", "false"} {
		args := workerArgs(t, map[string]string{"name": "shop", "namespace": "default", "celExpressions": v})
		assert.Contains(t, args, "--enable-cel-expressions="+v)
	}
}
