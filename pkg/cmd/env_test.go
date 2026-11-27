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
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv unsets every flag's variable for the test, so nothing the test
// process inherits leaks in. Each test calls it before setting its own.
func clearEnv(t *testing.T) {
	fs := pflag.NewFlagSet("kubetrigger", pflag.ContinueOnError)
	addFlags(newOption(), fs)
	fs.VisitAll(func(f *pflag.Flag) {
		t.Setenv(envName(f.Name), "")
		require.NoError(t, os.Unsetenv(envName(f.Name)))
	})
}

// parse reads args into a fresh option, then applies the environment.
func parse(t *testing.T, args ...string) (*option, error) {
	o := newOption()
	fs := pflag.NewFlagSet("kubetrigger", pflag.ContinueOnError)
	addFlags(o, fs)
	require.NoError(t, fs.Parse(args))
	_, err := applyEnv(fs)
	return o, err
}

func TestEnvSetsAFlagNotGiven(t *testing.T) {
	clearEnv(t)
	t.Setenv("MAX_RETRY", "9")
	o, err := parse(t)
	require.NoError(t, err)
	assert.Equal(t, 9, o.MaxRetry)
}

func TestFlagBeatsEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("MAX_RETRY", "9")
	o, err := parse(t, "--max-retry=3")
	require.NoError(t, err)
	assert.Equal(t, 3, o.MaxRetry)
}

func TestDefaultWithoutEither(t *testing.T) {
	clearEnv(t)
	o, err := parse(t)
	require.NoError(t, err)
	assert.Equal(t, defaultMaxRetry, o.MaxRetry)
}

func TestEnvNames(t *testing.T) {
	clearEnv(t)
	t.Setenv("PER_WORKER_QPS", "7")
	t.Setenv("LEADER_ELECT", "true")
	t.Setenv("LEADER_ELECTION_LEASE_DURATION", "42s")
	o, err := parse(t)
	require.NoError(t, err)
	assert.Equal(t, 7, o.PerWorkerQPS)
	assert.True(t, enableLeaderElection)
	assert.Equal(t, 42*time.Second, leaseDuration)
}

func TestBadEnvValueNamesItsVariable(t *testing.T) {
	clearEnv(t)
	t.Setenv("MAX_RETRY", "lots")
	_, err := parse(t)
	assert.ErrorContains(t, err, "MAX_RETRY")
}

func TestValidationNamesTheVariable(t *testing.T) {
	clearEnv(t)
	for name, env := range map[string]map[string]string{
		"a number out of range": {"WORKERS": "0"},
		"a bad log level":       {"LOG_LEVEL": "bogus"},
	} {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			c := NewCommand()
			var variable string
			for k, v := range env {
				t.Setenv(k, v)
				variable = k
			}
			require.NoError(t, c.ParseFlags(nil))
			assert.ErrorContains(t, c.PreRunE(c, nil), variable)
		})
	}
}

func TestActionRetry(t *testing.T) {
	clearEnv(t)
	o, err := parse(t, "--action-retry")
	require.NoError(t, err)
	assert.True(t, o.ActionRetry)

	t.Setenv("ACTION_RETRY", "true")
	o, err = parse(t)
	require.NoError(t, err)
	assert.True(t, o.ActionRetry)
}
