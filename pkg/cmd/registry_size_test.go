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

package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --registry-size sized a cache that no longer exists. It is still accepted,
// so existing command lines keep working, but it is marked as doing nothing.
func TestRegistrySizeIsAcceptedButDeprecated(t *testing.T) {
	clearEnv(t)
	o := newOption()
	fs := pflag.NewFlagSet("kubetrigger", pflag.ContinueOnError)
	addFlags(o, fs)

	require.NoError(t, fs.Parse([]string{"--registry-size=0"}))

	assert.NotEmpty(t, fs.Lookup(FlagRegistrySize).Deprecated)
	assert.NoError(t, o.validate(), "a flag with no effect must not fail validation")
}
