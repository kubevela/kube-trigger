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

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeEvents(t *testing.T) {
	testcases := map[string]struct {
		first, second []EventType
		merged        []EventType
	}{
		"different events are combined": {
			first:  []EventType{EventTypeCreate},
			second: []EventType{EventTypeUpdate},
			merged: []EventType{EventTypeCreate, EventTypeUpdate},
		},
		"overlapping events are listed once": {
			first:  []EventType{EventTypeCreate, EventTypeUpdate},
			second: []EventType{EventTypeUpdate, EventTypeDelete},
			merged: []EventType{EventTypeCreate, EventTypeUpdate, EventTypeDelete},
		},
		"no events means all, and stays all": {
			first:  nil,
			second: []EventType{EventTypeUpdate},
			merged: nil,
		},
		"a later trigger with no events widens to all": {
			first:  []EventType{EventTypeUpdate},
			second: nil,
			merged: nil,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			c := Config{Kind: "Deployment", Events: tc.first}
			c.Merge(Config{Kind: "Deployment", Events: tc.second})
			assert.Equal(t, tc.merged, c.Events)
		})
	}
}

func TestKeySeparatesWatchersByCluster(t *testing.T) {
	deployments := func(clusters ...string) *Config {
		return &Config{APIVersion: "apps/v1", Kind: "Deployment", Clusters: clusters}
	}

	testcases := map[string]struct {
		first, second *Config
		same          bool
	}{
		"different clusters are watched separately": {
			first:  deployments("cluster-a"),
			second: deployments("cluster-b"),
			same:   false,
		},
		"a subset of clusters is watched separately": {
			first:  deployments("cluster-a", "cluster-b"),
			second: deployments("cluster-a"),
			same:   false,
		},
		"the order of clusters does not matter": {
			first:  deployments("cluster-a", "cluster-b"),
			second: deployments("cluster-b", "cluster-a"),
			same:   true,
		},
		"a cluster named twice counts once": {
			first:  deployments("cluster-a", "cluster-a"),
			second: deployments("cluster-a"),
			same:   true,
		},
		"no clusters means the local cluster": {
			first:  deployments(),
			second: deployments("local"),
			same:   true,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			if tc.same {
				assert.Equal(t, tc.first.Key(), tc.second.Key())
			} else {
				assert.NotEqual(t, tc.first.Key(), tc.second.Key())
			}
		})
	}
}
