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

// Package expression evaluates $( ) CEL expressions over the context a
// trigger's filter and action see, using the engine in kubevela/pkg/cel.
package expression

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubevela/pkg/cel"
	"github.com/kubevela/pkg/cel/template"
	"k8s.io/apimachinery/pkg/runtime"
	utiljson "k8s.io/apimachinery/pkg/util/json"
)

// Root is the one name expressions read: the context filters and actions see,
// so a CEL expression reads the same paths a CUE filter does.
const Root = "context"

// engine is immutable and safe for concurrent use, so every trigger shares it.
var engine = func() *cel.Engine {
	e, err := cel.NewEngine(cel.Root{Name: Root})
	if err != nil {
		panic(err)
	}
	return e
}()

// filterKey is where a filter sits in the one-property tree it is planned as.
const filterKey = "filter"

// Filter is a CEL filter, planned once and evaluated per event.
type Filter struct {
	plan *cel.Plan
}

// NewFilter plans filter if it is CEL: exactly one $( ) expression, with
// nothing around it but space. It reports false for any other filter, which is
// CUE. A CEL filter that does not parse or compile is an error, including one
// that opens with $( and never closes.
func NewFilter(filter string) (*Filter, bool, error) {
	trimmed := strings.TrimSpace(filter)
	parsed, err := template.Parse(trimmed)
	if err != nil {
		if strings.HasPrefix(trimmed, "$(") {
			return nil, true, err
		}
		return nil, false, nil
	}
	if len(parsed.Fragments) != 1 || !parsed.Fragments[0].IsExpr() {
		return nil, false, nil
	}
	plan, err := engine.Plan(map[string]interface{}{filterKey: trimmed})
	if err != nil {
		return nil, true, err
	}
	return &Filter{plan: plan}, true, nil
}

// Eval reports whether the event is kept. A result that is not a bool is an
// error, as is reading a field that is not there; has() tests for one.
func (f *Filter) Eval(ctx context.Context, contextData map[string]interface{}) (bool, error) {
	out, err := eval(ctx, f.plan, contextData)
	if err != nil {
		return false, err
	}
	kept, ok := out.(map[string]interface{})[filterKey].(bool)
	if !ok {
		return false, fmt.Errorf("filter must evaluate to a bool, got %T", out.(map[string]interface{})[filterKey])
	}
	return kept, nil
}

// Properties are an action's properties with their $( ) expressions planned.
type Properties struct {
	plan *cel.Plan
}

// NewProperties plans every $( ) expression in props. One that does not
// compile is an error naming its property; $$( is a literal $(.
func NewProperties(props *runtime.RawExtension) (*Properties, error) {
	if props == nil || len(props.Raw) == 0 {
		return &Properties{}, nil
	}
	plan, err := engine.PlanJSON(props.Raw)
	if err != nil {
		return nil, fmt.Errorf("%w; write a literal $( as $$(", err)
	}
	return &Properties{plan: plan}, nil
}

// Eval renders the properties for one event. An expression that is a
// property's whole value keeps its type.
func (p *Properties) Eval(ctx context.Context, contextData map[string]interface{}) (*runtime.RawExtension, error) {
	if p.plan == nil {
		return nil, nil
	}
	out, err := eval(ctx, p.plan, contextData)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return &runtime.RawExtension{Raw: raw}, nil
}

func eval(ctx context.Context, plan *cel.Plan, contextData map[string]interface{}) (interface{}, error) {
	value, err := jsonShaped(contextData)
	if err != nil {
		return nil, err
	}
	return plan.Eval(ctx, map[string]cel.Resolver{Root: cel.Static(value)}, cel.TreeOptions{})
}

// jsonShaped turns the context, which holds objects and structs, into the
// decoded-document shape a resolver delivers. Whole numbers stay int64, as in
// an unstructured object.
func jsonShaped(contextData map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(contextData)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{}
	return out, utiljson.Unmarshal(b, &out)
}
