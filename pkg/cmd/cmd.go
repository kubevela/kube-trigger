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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	v1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubevela/kube-trigger/pkg/config"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	"github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/kubevela/kube-trigger/pkg/util/client"
	"github.com/kubevela/kube-trigger/pkg/version"
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

	FlagLeaderElect                 = "leader-elect"
	FlagLeaderElectionLeaseDuration = "leader-election-lease-duration"
	FlagLeaderElectionRenewDeadline = "leader-election-renew-deadline"
	FlagLeaderElectionRetryPeriod   = "leader-election-retry-period"
)

const (
	cmdLongHelp = `kube-trigger can watch events and run actions accordingly.

All command-line options can be specified as environment variables, which are defined by the command-line option, 
capitalized, with all -’s replaced with _’s.

For example, $LOG_LEVEL can be used in place of --log-level

Options have a priority like this: cli-flags > env > default-values`
)

var (
	logger               = logrus.WithField("kubetrigger", "main")
	opt                  = newOption()
	enableLeaderElection bool

	leaseDuration time.Duration
	renewDeadline time.Duration
	retryPeriod   time.Duration
)

// NewCommand news a command
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:  "kubetrigger",
		Long: cmdLongHelp,
		RunE: runCli,
		PreRunE: func(_ *cobra.Command, _ []string) error {
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
		Run: func(_ *cobra.Command, _ []string) {
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
	f.BoolVar(&enableLeaderElection, FlagLeaderElect, false, "Enable leader election for kube-trigger. Enabling this will ensure there is only one active kube-trigger.")
	f.DurationVar(&leaseDuration, FlagLeaderElectionLeaseDuration, defaultLeaseDuration, "The duration that non-leader candidates will wait to force acquire leadership.")
	f.DurationVar(&renewDeadline, FlagLeaderElectionRenewDeadline, defaultRenewDeadline, "The duration that the acting controlplane will retry refreshing leadership before giving up.")
	f.DurationVar(&retryPeriod, FlagLeaderElectionRetryPeriod, defaultRetryPeriod, "The duration the LeaderElector clients should wait between tries of actions.")
}

func runCli(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	r := NewRunner()
	if err := r.Start(ctx); err != nil {
		return err
	}
	// Listen to termination signals.
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGINT)
	select {
	case err := <-r.Err():
		logger.Errorf("runner stop with err: %v", err)
		return err
	case <-ctx.Done():
		logger.Infof("context cancelled, stopping")
	case <-sigterm:
		logger.Infof("received termination signal, stopping")
	}
	return nil
}

func run(ctx context.Context) error {
	// Set log level. No need to check error, we validated it previously.
	level, _ := logrus.ParseLevel(opt.LogLevel)
	logrus.SetLevel(level)

	cli, err := client.GetClient()
	if err != nil {
		return err
	}

	// Create registries for Sources
	sourceReg := sourceregistry.NewWithBuiltinSources()

	// Get our kube-trigger config.
	conf, err := config.NewFromFileOrDir(opt.Config)
	if err != nil {
		return errors.Wrapf(err, "error when parsing config %s", opt.Config)
	}
	err = conf.Validate(ctx, cli, sourceReg)
	if err != nil {
		return errors.Wrapf(err, "cannot validate config")
	}

	defer utilruntime.HandleCrash()

	// Create an executor for running Action jobs.
	exe, err := executor.New(opt.getExecutorConfig())
	if err != nil {
		return errors.Wrap(err, "error when creating executor")
	}

	instances := make(map[string]types.Source)

	// Run watchers.
	for _, w := range conf.Triggers {
		// Make this Source type exists.
		s, ok := sourceReg.Get(w.Source.Type)
		if !ok {
			logger.Errorf("source type %s does not exist", w.Source.Type)
			continue
		}

		source := s.New()
		if s.Singleton() {
			if s, ok := instances[w.Source.Type]; ok {
				source = s
			}
		}

		// Create a EventHandler
		eh := eventhandler.NewFromConfig(ctx, cli, w.Action, w.Filter, exe)

		// Initialize Source, with user-provided prop and event handler
		err = source.Init(w.Source.Properties, eh)
		if err != nil {
			logger.Errorf("failed to initialize source %s: %s", source.Type(), err)
			continue
		}

		instances[w.Source.Type] = source
	}

	for _, instance := range instances {
		err := instance.Run(ctx)
		if err != nil {
			logger.Errorf("source %s failed to run: %v", instance.Type(), err)
			continue
		}
	}

	// Let the workers run Actions.
	exe.RunJobs(ctx)
	return nil
}

// Runner manages the task execution.
type Runner struct {
	errChan chan error
	start   func(ctx context.Context) error
	once    sync.Once
}

// NewRunner new a Runner
func NewRunner() *Runner {
	return &Runner{
		errChan: make(chan error),
		start:   run,
	}
}

// Start the task.
func (r *Runner) Start(ctx context.Context) error {
	if enableLeaderElection {
		return r.startLeaderElection(ctx)
	}
	go func() {
		if err := r.start(ctx); err != nil {
			r.errChan <- err
		}
	}()
	return nil
}

func (r *Runner) startLeaderElection(ctx context.Context) error {
	cclient, err := v1.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return err
	}
	leaderElectionName := fmt.Sprintf("kube-trigger-%s",
		strings.ToLower(strings.ReplaceAll(version.Version, ".", "-")),
	)
	leaderElectionID := fmt.Sprintf("%s-%s", leaderElectionName, uuid.NewUUID())
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaderElectionName,
			Namespace: "vela-system",
		},
		Client: cclient,
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: leaderElectionID,
		},
	}
	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			r.once.Do(func() {
				if err = r.start(ctx); err != nil {
					r.errChan <- err
				}
			})
		},
		OnStoppedLeading: func() {
			r.errChan <- errors.New("leader election lost")
		},
	}
	l, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          lock,
			LeaseDuration: leaseDuration,
			RenewDeadline: renewDeadline,
			RetryPeriod:   retryPeriod,
			Callbacks:     callbacks,
		})
	if err != nil {
		return err
	}
	go func() {
		l.Run(ctx)
	}()
	return err
}

// Err return the runner's runtime error
func (r *Runner) Err() chan error {
	return r.errChan
}
