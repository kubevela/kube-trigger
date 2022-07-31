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

package eventhandler

import (
	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	"github.com/kubevela/kube-trigger/pkg/executor"
)

// Store stores a list of EventHandlers.
type Store struct {
	ehs []EventHandler
}

// Add adds EventHandlers to store.
func (e *Store) Add(ehs ...EventHandler) {
	e.ehs = append(e.ehs, ehs...)
}

// Call calls ALL EventHandlers with the given arguments.
func (e *Store) Call(sourceType string, event interface{}) {
	for _, eh := range e.ehs {
		eh(sourceType, event)
	}
}

// NewStoreWithActionExecutors return a store that holds EventHandlers that
// will add a Job which will execute a given Action to Executor to be executed.
//
// It is commonly used to be passed to Source so that when an event happened,
// Source simply calls Store.Call() all Actions will be executed by Executor.
func NewStoreWithActionExecutors(
	exe *executor.Executor,
	reg *actionregistry.Registry,
	metas ...types.ActionMeta,
) Store {
	s := Store{}
	for _, m := range metas {
		s.Add(NewWithActionExecutor(exe, reg, m))
	}
	return s
}
