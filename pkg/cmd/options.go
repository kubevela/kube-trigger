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
	"time"

	"github.com/kubevela/kube-trigger/pkg/executor"
	"github.com/sirupsen/logrus"
)

type option struct {
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
	defaultTimeout      = 10

	defaultRegistrySize = 100

	// Values taken from: https://github.com/kubernetes/component-base/blob/master/config/v1alpha1/defaults.go
	defaultLeaseDuration = 15 * time.Second
	defaultRenewDeadline = 10 * time.Second
	defaultRetryPeriod   = 2 * time.Second
)

func newOption() *option {
	return &option{}
}

func (o *option) validate() error {
	_, err := logrus.ParseLevel(o.LogLevel)
	if err != nil {
		return err
	}
	if o.Config == "" {
		return fmt.Errorf("%s not specified", FlagConfig)
	}
	if o.QueueSize <= 0 {
		return fmt.Errorf("%s must be greater than 0", FlagQueueSize)
	}
	if o.Workers <= 0 {
		return fmt.Errorf("%s must be greater than 0", FlagWorkers)
	}
	if o.PerWorkerQPS <= 0 {
		return fmt.Errorf("%s must be greater than 0", FlagPerWorkerQPS)
	}
	if o.MaxRetry < 0 {
		return fmt.Errorf("%s must be greater or equal to 0", FlagMaxRetry)
	}
	if o.RetryDelay < 0 {
		return fmt.Errorf("%s must be greater or equal to 0", FlagRetryDelay)
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("%s must be greater than 0", FlagTimeout)
	}
	if o.RegistrySize <= 0 {
		return fmt.Errorf("%s must be greater than 0", FlagRegistrySize)
	}
	return nil
}

func (o *option) getExecutorConfig() executor.Config {
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
