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

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/config"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/executor"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	"github.com/kubevela/kube-trigger/pkg/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	FlagLogLevel    = "log-level"
	FlagConfig      = "config"
	FlagConfigShort = "c"

	FlagQueueSize    = "queue-size"
	FlagWorkers      = "workers"
	FlagPerWorkerQPS = "per-worker-qps"
	FlagMaxRetry     = "max-retry"
	FlagRetryDelay   = "retry-delay"
	FlagActionRetry  = "action-retry"
	FlagTimeout      = "timeout"

	FlagRegistrySize = "registry-size"
)

const (
	cmdLongHelp = `kube-trigger can watch events and run actions accordingly.

All command-line options can be specified as environment variables, which are defined by the command-line option, 
capitalized, with all -’s replaced with _’s.

For example, $LOG_LEVEL can be used in place of --log-level

Options have a priority like this: cli-flags > env > default-values`
)

func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:  "kubetrigger",
		Long: cmdLongHelp,
		RunE: runCli,
	}
	addFlags(c.Flags())
	c.AddCommand(NewVersionCommand())
	return c
}

func NewVersionCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "show kube-trigger version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			//nolint:forbidigo // print version
			fmt.Println(version.Version)
		},
	}
	return c
}

//nolint:lll
func addFlags(f *pflag.FlagSet) {
	f.StringP(FlagConfig, FlagConfigShort, defaultConfig, "Path to config file or directory. If a directory is provided, all files (recursively) will be appended together.")
	f.String(FlagLogLevel, defaultLogLevel, "Log level")
	f.Int(FlagQueueSize, defaultQueueSize, "Queue size for running actions, this is shared between all watchers")
	f.Int(FlagWorkers, defaultWorkers, "Number of workers for running actions, this is shared between all watchers")
	f.Int(FlagPerWorkerQPS, defaultPerWorkerQPS, "Long-term QPS limiting per worker, this is shared between all watchers")
	f.Bool(FlagActionRetry, defaultActionRetry, "Retry actions if it fails")
	f.Int(FlagMaxRetry, defaultMaxRetry, "Retry count after action failed, valid only when action retrying is enabled")
	f.Int(FlagRetryDelay, defaultRetryDelay, "First delay to retry actions in seconds, subsequent delay will grow exponentially")
	f.Int(FlagTimeout, defaultTimeout, "Timeout for running each action")
	f.Int(FlagRegistrySize, defaultRegistrySize, "Cache size for filters and actions")
}

var logger = logrus.WithField("kubetrigger", "main")

func runCli(cmd *cobra.Command, args []string) error {
	var err error

	// Read options from env and cli, and fall back to defaults.
	opt, err := NewOption().
		WithDefaults().
		WithEnvVariables().
		WithCliFlags(cmd.Flags()).
		Validate()
	if err != nil {
		return errors.Wrap(err, "error when paring flags")
	}

	// Set log level. No need to check error, we validated it previously.
	level, _ := logrus.ParseLevel(opt.LogLevel)
	logrus.SetLevel(level)

	// Get our kube-trigger config.
	conf, err := config.NewFromFileOrDir(opt.Config)
	if err != nil {
		return errors.Wrapf(err, "error when parsing config %s", opt.Config)
	}

	// Create registries for Sources, Filers, and Actions.
	sourceReg := sourceregistry.NewWithBuiltinSources()
	filterReg := filterregistry.NewWithBuiltinFilters(opt.RegistrySize)
	actionReg := actionregistry.NewWithBuiltinActions(opt.RegistrySize)

	defer utilruntime.HandleCrash()

	// Create an executor for running Action jobs.
	exe, err := executor.New(opt.GetExecutorConfig())
	if err != nil {
		return errors.Wrap(err, "error when creating executor")
	}
	defer exe.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run watchers.
	for _, w := range conf.Watchers {
		// Make this Source type exists.
		s, ok := sourceReg.Get(w.Source)
		if !ok {
			return fmt.Errorf("source type %s does not exist", w.Source.Type)
		}

		// New Source instance.
		ns := s.New()

		c := eventhandler.Config{
			Filters:        w.Filters,
			Actions:        w.Actions,
			FilterRegistry: filterReg,
			ActionRegistry: actionReg,
			Executor:       exe,
		}
		// Create a EventHandler
		eh := eventhandler.NewFromConfig(c)

		// Initialize Source, with user-provided prop and event handler
		err = ns.Init(w.Source.Properties, eh)
		if err != nil {
			return errors.Wrapf(err, "failed to initialize source %s", ns.Type())
		}

		// Time to run this source.
		go func() {
			//nolint:govet // this err-shadowing fine
			err := ns.Run(ctx)
			if err != nil {
				logger.Fatalf("source %s failed to run", ns.Type())
				return
			}
		}()
	}

	// Let the workers run Actions.
	exe.RunJobs(ctx)

	// Listen to termination signals.
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	select {
	case <-ctx.Done():
		logger.Infof("context cancelled, stopping")
	case <-sigterm:
		logger.Infof("recived termination signel, stopping")
	}

	return nil
}
