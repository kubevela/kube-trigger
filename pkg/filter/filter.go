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

package filter

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/tools/fix"
	"github.com/kubevela/pkg/cue/cuex"
)

// ApplyFilter applies the given filter to an object.
func ApplyFilter(ctx context.Context, contextData map[string]interface{}, filter string) (bool, error) {
	template, err := BuildFilterTemplate(filter)
	if err != nil {
		return false, err
	}
	filterVal, err := cuex.CompileStringWithOptions(ctx, template, cuex.WithExtraData("context", contextData))
	if err != nil {
		return false, err
	}
	if filterVal.Err() != nil {
		return false, filterVal.Err()
	}
	result := filterVal.LookupPath(cue.ParsePath("filter"))
	if filterVal.LookupPath(cue.ParsePath("filter.filter")).Exists() {
		result = filterVal.LookupPath(cue.ParsePath("filter.filter"))
	}
	if result.Err() != nil {
		return false, result.Err()
	}
	resultBool, err := result.Bool()
	// if the result is not a bool, return true to pass the filter
	if err != nil {
		return true, nil
	}
	return resultBool, nil
}

// BuildFilterTemplate build filter template
func BuildFilterTemplate(filter string) (string, error) {
	f, err := parser.ParseFile("-", filter)
	if err != nil {
		return "", err
	}
	n := fix.File(f)
	if n.Imports == nil {
		return fmt.Sprintf("filter: %s", filter), nil
	}
	var importDecls, contentDecls []ast.Decl
	for _, decl := range n.Decls {
		if importDecl, ok := decl.(*ast.ImportDecl); ok {
			importDecls = append(importDecls, importDecl)
		} else {
			contentDecls = append(contentDecls, decl)
		}
	}
	importString, err := encodeDeclsToString(importDecls)
	if err != nil {
		return "", err
	}
	contentString, err := encodeDeclsToString(contentDecls)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(filterTemplate, importString, contentString), nil
}

func encodeDeclsToString(decls []ast.Decl) (string, error) {
	bs, err := format.Node(&ast.File{Decls: decls}, format.Simplify())
	if err != nil {
		return "", fmt.Errorf("failed to encode cue: %w", err)
	}
	return strings.TrimSpace(string(bs)), nil
}

var filterTemplate = `
%s
filter: {
	%s
}
`
