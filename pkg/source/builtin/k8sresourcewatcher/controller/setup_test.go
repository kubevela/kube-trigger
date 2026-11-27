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

package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
)

var imageKind = schema.GroupVersionKind{Group: "kpack.io", Version: "v1alpha2", Kind: "Image"}

// lateMapper does not know imageKind until it has been asked for it `missing`
// times, as when the CRD is installed after kube-trigger starts.
type lateMapper struct {
	meta.RESTMapper
	missing int32
	asked   atomic.Int32
	// err is returned while the kind is missing; a no-match error if nil.
	err error
}

func newLateMapper(missing int32) *lateMapper {
	known := meta.NewDefaultRESTMapper(nil)
	known.Add(imageKind, meta.RESTScopeNamespace)
	return &lateMapper{RESTMapper: known, missing: missing}
}

func (m *lateMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	if m.asked.Add(1) <= m.missing {
		if m.err != nil {
			return nil, m.err
		}
		return nil, &meta.NoKindMatchError{GroupKind: gk, SearchedVersions: versions}
	}
	return m.RESTMapper.RESTMapping(gk, versions...)
}

func TestSetupReportsAKindThatIsNotInstalled(t *testing.T) {
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mapper := meta.NewDefaultRESTMapper(nil)

	c, err := Setup(context.Background(), cli, mapper, types.Config{APIVersion: "kpack.io/v1alpha2", Kind: "Image"}, nil)

	assert.Nil(t, c)
	assert.True(t, meta.IsNoMatchError(err), "want a no-match error, got %v", err)
}

func TestSetupReportsAnAPIVersionThatDoesNotParse(t *testing.T) {
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	c, err := Setup(context.Background(), cli, meta.NewDefaultRESTMapper(nil), types.Config{APIVersion: "a/b/c", Kind: "Image"}, nil)

	assert.Nil(t, c)
	assert.Error(t, err)
}

func TestSetupWhenInstalledWaitsForTheKind(t *testing.T) {
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mapper := newLateMapper(2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := SetupWhenInstalled(ctx, cli, mapper, types.Config{APIVersion: "kpack.io/v1alpha2", Kind: "Image"}, nil, 10*time.Millisecond)

	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, int32(3), mapper.asked.Load())
}

func TestSetupWhenInstalledStopsWithItsContext(t *testing.T) {
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c, err := SetupWhenInstalled(ctx, cli, meta.NewDefaultRESTMapper(nil), types.Config{APIVersion: "kpack.io/v1alpha2", Kind: "Image"}, nil, 10*time.Millisecond)

	assert.Nil(t, c)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSetupWhenInstalledDoesNotRetryABadAPIVersion(t *testing.T) {
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mapper := newLateMapper(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := SetupWhenInstalled(ctx, cli, mapper, types.Config{APIVersion: "a/b/c", Kind: "Image"}, nil, 10*time.Millisecond)

	assert.Nil(t, c)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(0), mapper.asked.Load())
}

// entriesAt counts the entries logged at level since the hook was installed.
func entriesAt(hook *logrustest.Hook, level logrus.Level) int {
	n := 0
	for _, e := range hook.AllEntries() {
		if e.Level == level {
			n++
		}
	}
	return n
}

func TestSetupWhenInstalledRetriesAMapperThatFails(t *testing.T) {
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mapper := newLateMapper(2)
	mapper.err = errors.New("discovery failed: the server is currently unable to handle the request")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := SetupWhenInstalled(ctx, cli, mapper, types.Config{APIVersion: "kpack.io/v1alpha2", Kind: "Image"}, nil, 10*time.Millisecond)

	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, int32(3), mapper.asked.Load())
}

func TestSetupWhenInstalledWarnsOnceForTheSameError(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mapper := newLateMapper(4)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := SetupWhenInstalled(ctx, cli, mapper, types.Config{APIVersion: "kpack.io/v1alpha2", Kind: "Image"}, nil, 10*time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, 1, entriesAt(hook, logrus.WarnLevel), "four misses with the same error warn once")
}

func TestRunWhenInstalledIsQuietWhenStopped(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	RunWhenInstalled(ctx, cli, meta.NewDefaultRESTMapper(nil), types.Config{APIVersion: "kpack.io/v1alpha2", Kind: "Image"}, nil, 10*time.Millisecond)

	assert.Equal(t, 0, entriesAt(hook, logrus.ErrorLevel), "stopping while waiting for a kind is not an error")
}

func TestRunWhenInstalledLogsAWatcherItCannotSetUp(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()
	cli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	RunWhenInstalled(context.Background(), cli, meta.NewDefaultRESTMapper(nil), types.Config{APIVersion: "a/b/c", Kind: "Image"}, nil, 10*time.Millisecond)

	assert.Equal(t, 1, entriesAt(hook, logrus.ErrorLevel))
}
