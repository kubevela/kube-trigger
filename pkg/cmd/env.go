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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// applyEnv sets each flag not given on the command line from its environment
// variable, so the order is command line, then environment, then default. It
// returns the variables it applied.
func applyEnv(fs *pflag.FlagSet) ([]string, error) {
	var applied []string
	var errs []error
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed || f.Name == "help" {
			return
		}
		name := envName(f.Name)
		v, ok := os.LookupEnv(name)
		if !ok {
			return
		}
		if err := fs.Set(f.Name, v); err != nil {
			errs = append(errs, fmt.Errorf("$%s: %w", name, err))
			return
		}
		applied = append(applied, "$"+name)
	})
	return applied, errors.Join(errs...)
}

// envName is a flag's environment variable: its name in capitals, with - as _.
func envName(flag string) string {
	return strings.ToUpper(strings.ReplaceAll(flag, "-", "_"))
}
