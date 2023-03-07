/*
Copyright 2023 The KubeVela Authors.

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

package templates

import (
	"context"
	"embed"
	"fmt"

	"github.com/kubevela/pkg/util/template/definition"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utilclient "github.com/kubevela/kube-trigger/pkg/util/client"
)

var (
	//go:embed static/*
	templateFS embed.FS
)

const (
	templateDir = "static"
)

// Loader is the interface for loading templates.
type Loader interface {
	LoadTemplate(context.Context, string) (string, error)
}

// GenericLoader is a loader that loads templates for sources, filters and actions
type GenericLoader struct {
	typ string
	cli client.Client
}

// NewLoader creates a new loader for the given type.
func NewLoader(t string) (*GenericLoader, error) {
	cli, err := utilclient.GetClient()
	if err != nil {
		return nil, err
	}
	return &GenericLoader{typ: t, cli: *cli}, nil
}

// LoadTemplate loads a template by name.
func (l *GenericLoader) LoadTemplate(ctx context.Context, name string) (string, error) {
	files, err := templateFS.ReadDir(fmt.Sprintf("%s/%s", templateDir, l.typ))
	if err != nil {
		return "", err
	}

	staticFilename := name + ".cue"
	for _, file := range files {
		if staticFilename == file.Name() {
			fileName := fmt.Sprintf("%s/%s/%s", templateDir, l.typ, file.Name())
			content, err := templateFS.ReadFile(fileName)
			return string(content), err
		}
	}

	template, err := definition.NewTemplateLoader(ctx, l.cli).LoadTemplate(ctx, name, definition.WithType(l.typ))
	if err != nil {
		return "", err
	}
	return template.Compile(), nil
}
