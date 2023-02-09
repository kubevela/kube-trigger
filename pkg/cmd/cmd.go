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

	"github.com/kubevela/kube-trigger/pkg/config"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	"github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/kubevela/kube-trigger/pkg/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// nolint:revive
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

var (
	logger = logrus.WithField("kubetrigger", "main")
	opt    = newOption()
)

// NewCommand news a command
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:  "kubetrigger",
		Long: cmdLongHelp,
		RunE: runCli,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			logger.Infof("kube-trigger version=%s", version.Version)
			return nil
		},
	}
	c.AddCommand(newVersionCommand())
	addFlags(opt, c.Flags())
	if err := opt.validate(); err != nil {
		panic(err)
	}
	return c
}

func newVersionCommand() *cobra.Command {
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
func addFlags(opt *option, f *pflag.FlagSet) {
	f.StringVarP(&opt.Config, FlagConfig, FlagConfigShort, defaultConfig, "Path to config file or directory. If a directory is provided, all files inside that directory will be combined together. Supported file formats are: json, yaml, and cue.")
	f.StringVar(&opt.LogLevel, FlagLogLevel, defaultLogLevel, "Log level")
	f.IntVar(&opt.QueueSize, FlagQueueSize, defaultQueueSize, "Queue size for running actions, this is shared between all watchers")
	f.IntVar(&opt.Workers, FlagWorkers, defaultWorkers, "Number of workers for running actions, this is shared between all watchers")
	f.IntVar(&opt.PerWorkerQPS, FlagPerWorkerQPS, defaultPerWorkerQPS, "Long-term QPS limiting per worker, this is shared between all watchers")
	f.IntVar(&opt.MaxRetry, FlagMaxRetry, defaultMaxRetry, "Retry count after action failed, valid only when action retrying is enabled")
	f.IntVar(&opt.RetryDelay, FlagRetryDelay, defaultRetryDelay, "First delay to retry actions in seconds, subsequent delay will grow exponentially")
	f.IntVar(&opt.Timeout, FlagTimeout, defaultTimeout, "Timeout for running each action")
	f.IntVar(&opt.RegistrySize, FlagRegistrySize, defaultRegistrySize, "Cache size for filters and actions")
	f.StringVar(&k8sresourcewatcher.MultiClusterConfigType, "multi-cluster-config-type", k8sresourcewatcher.TypeClusterGateway, "Multi-cluster config type, supported types: cluster-gateway, cluster-gateway-kubeconfig")
}

func runCli(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	// Set log level. No need to check error, we validated it previously.
	level, _ := logrus.ParseLevel(opt.LogLevel)
	logrus.SetLevel(level)

	// Create registries for Sources
	sourceReg := sourceregistry.NewWithBuiltinSources()

	// Get our kube-trigger config.
	conf, err := config.NewFromFileOrDir(opt.Config)
	if err != nil {
		return errors.Wrapf(err, "error when parsing config %s", opt.Config)
	}
	err = conf.Validate(ctx, sourceReg)
	if err != nil {
		return errors.Wrapf(err, "cannot validate config")
	}

	defer utilruntime.HandleCrash()

	// Create an executor for running Action jobs.
	exe, err := executor.New(opt.getExecutorConfig())
	if err != nil {
		return errors.Wrap(err, "error when creating executor")
	}
	defer exe.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instances := make(map[string]types.Source)

	// Run watchers.
	for _, w := range conf.Triggers {
		// Make this Source type exists.
		s, ok := sourceReg.Get(w.Source.Type)
		if !ok {
			return fmt.Errorf("source type %s does not exist", w.Source.Type)
		}
		source := s.New()
		if s, ok := instances[w.Source.Type]; ok {
			source = s
		}

		// Create a EventHandler
		eh := eventhandler.NewFromConfig(ctx, w.Action, w.Filter, exe)

		// Initialize Source, with user-provided prop and event handler
		err = source.Init(w.Source.Properties, eh)
		if err != nil {
			return errors.Wrapf(err, "failed to initialize source %s", source.Type())
		}

		instances[w.Source.Type] = source
	}

	for _, instance := range instances {
		err := instance.Run(ctx)
		if err != nil {
			logger.Fatalf("source %s failed to run: %v", instance.Type(), err)
			return err
		}
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
		logger.Infof("received termination signal, stopping")
	}

	return nil
}
