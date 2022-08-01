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
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type Option struct {
	LogLevel string
	Config   string

	QueueSize    int
	Workers      int
	PerWorkerQPS int
	MaxRetry     int
	RetryDelay   int
	ActionRetry  bool
	Timeout      int

	RegistrySize int
}

const (
	defaultLogLevel = "info"
	defaultConfig   = "config.cue"

	defaultQueueSize    = 50
	defaultWorkers      = 4
	defaultPerWorkerQPS = 2
	defaultMaxRetry     = 5
	defaultRetryDelay   = 2
	defaultActionRetry  = false
	defaultTimeout      = 5

	defaultRegistrySize = 100
)

const (
	envStrLogLevel = "LOG_LEVEL"
	envStrConfig   = "CONFIG"

	envStrQueueSize    = "QUEUE_SIZE"
	envStrWorkers      = "WORKERS"
	envStrPerWorkerQPS = "PER_WORKER_QPS"
	envStrMaxRetry     = "MAX_RETRY"
	envStrRetryDelay   = "RETRY_DELAY"
	envStrActionRetry  = "ACTION_RETRY"
	envStrTimeout      = "TIMEOUT"

	envStrRegistrySize = "REGISTRY_SIZE"
)

func NewOption() *Option {
	return &Option{}
}

func (o *Option) WithDefaults() *Option {
	o.LogLevel = defaultLogLevel
	o.Config = defaultConfig
	o.QueueSize = defaultQueueSize
	o.Workers = defaultWorkers
	o.PerWorkerQPS = defaultPerWorkerQPS
	o.MaxRetry = defaultMaxRetry
	o.RetryDelay = defaultRetryDelay
	o.ActionRetry = defaultActionRetry
	o.Timeout = defaultTimeout
	o.RegistrySize = defaultRegistrySize
	return o
}

//nolint:gocognit
func (o *Option) WithEnvVariables() *Option {
	if v, ok := os.LookupEnv(envStrLogLevel); ok && v != "" {
		o.LogLevel = v
	}
	if v, ok := os.LookupEnv(envStrConfig); ok && v != "" {
		o.Config = v
	}
	if v, ok := os.LookupEnv(envStrQueueSize); ok && v != "" {
		o.QueueSize, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv(envStrWorkers); ok && v != "" {
		o.Workers, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv(envStrPerWorkerQPS); ok && v != "" {
		o.PerWorkerQPS, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv(envStrMaxRetry); ok && v != "" {
		o.MaxRetry, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv(envStrRetryDelay); ok && v != "" {
		o.RetryDelay, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv(envStrActionRetry); ok && v != "" {
		o.ActionRetry, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv(envStrTimeout); ok && v != "" {
		o.Timeout, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv(envStrRegistrySize); ok && v != "" {
		o.RegistrySize, _ = strconv.Atoi(v)
	}
	return o
}

//nolint:gocognit
func (o *Option) WithCliFlags(flags *pflag.FlagSet) *Option {
	if v, err := flags.GetString(flagLogLevel); err == nil && flags.Changed(flagLogLevel) {
		o.LogLevel = v
	}
	if v, err := flags.GetString(flagConfig); err == nil && flags.Changed(flagConfig) {
		o.Config = v
	}
	if v, err := flags.GetInt(flagQueueSize); err == nil && flags.Changed(flagQueueSize) {
		o.QueueSize = v
	}
	if v, err := flags.GetInt(flagWorkers); err == nil && flags.Changed(flagWorkers) {
		o.Workers = v
	}
	if v, err := flags.GetInt(flagPerWorkerQPS); err == nil && flags.Changed(flagPerWorkerQPS) {
		o.PerWorkerQPS = v
	}
	if v, err := flags.GetInt(flagMaxRetry); err == nil && flags.Changed(flagMaxRetry) {
		o.MaxRetry = v
	}
	if v, err := flags.GetInt(flagRetryDelay); err == nil && flags.Changed(flagRetryDelay) {
		o.RetryDelay = v
	}
	if v, err := flags.GetBool(flagActionRetry); err == nil && flags.Changed(flagActionRetry) {
		o.ActionRetry = v
	}
	if v, err := flags.GetInt(flagTimeout); err == nil && flags.Changed(flagTimeout) {
		o.Timeout = v
	}
	if v, err := flags.GetInt(flagRegistrySize); err == nil && flags.Changed(flagRegistrySize) {
		o.RegistrySize = v
	}
	return o
}

func (o *Option) Validate() (*Option, error) {
	_, err := logrus.ParseLevel(o.LogLevel)
	if err != nil {
		return nil, err
	}
	if o.Config == "" {
		return nil, fmt.Errorf("%s not specified", flagConfig)
	}
	if o.QueueSize <= 0 {
		return nil, fmt.Errorf("%s must be greater than 0", flagQueueSize)
	}
	if o.Workers <= 0 {
		return nil, fmt.Errorf("%s must be greater than 0", flagWorkers)
	}
	if o.PerWorkerQPS <= 0 {
		return nil, fmt.Errorf("%s must be greater than 0", flagPerWorkerQPS)
	}
	if o.MaxRetry < 0 {
		return nil, fmt.Errorf("%s must be greater or equal to 0", flagMaxRetry)
	}
	if o.RetryDelay < 0 {
		return nil, fmt.Errorf("%s must be greater or equal to 0", flagRetryDelay)
	}
	if o.Timeout <= 0 {
		return nil, fmt.Errorf("%s must be greater than 0", flagTimeout)
	}
	if o.RegistrySize <= 0 {
		return nil, fmt.Errorf("%s must be greater than 0", flagRegistrySize)
	}
	return o, nil
}

func (o *Option) GetExecutorConfig() executor.Config {
	return executor.Config{
		QueueSize:            o.QueueSize,
		Workers:              o.Workers,
		MaxJobRetries:        o.MaxRetry,
		BaseRetryDelay:       time.Second * time.Duration(o.RetryDelay),
		RetryJobAfterFailure: o.ActionRetry,
		PerWorkerQPS:         o.PerWorkerQPS,
		Timeout:              time.Second * time.Duration(o.Timeout),
	}
}
