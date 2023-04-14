/*
Copyright 2023 The KubeVela Authors.

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

package cronjob

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubevela/kube-trigger/pkg/eventhandler"
)

func TestCronJob_Init(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "normal",
			config: Config{
				Schedule: "* * * * *",
				TimeZone: "Asia/Shanghai",
			},
			wantErr: false,
		},
		{
			name: "invalid_schedule",
			config: Config{
				Schedule: "0 0 0 0 0",
				TimeZone: "",
			},
			wantErr: true,
		},
		{
			name: "invalid_timezone",
			config: Config{
				Schedule: "* * * * *",
				TimeZone: "Nowhere/nowhere",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := (&CronJob{}).New()
			re := &runtime.RawExtension{}
			b, err := json.Marshal(tt.config)
			if err != nil {
				t.Fail()
			}
			err = re.UnmarshalJSON(b)
			if err != nil {
				t.Fail()
			}
			if err := c.Init(re, eventhandler.New()); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("invalid_properties", func(t *testing.T) {
		c := (&CronJob{}).New()
		re := &runtime.RawExtension{
			Raw:    []byte("this-is-not-valid"),
			Object: nil,
		}
		err := c.Init(re, eventhandler.New())
		assert.Error(t, err)
	})
}

func TestCronJob_Run(t *testing.T) {
	c := CronJob{
		config:     Config{},
		cronRunner: cron.New(),
	}
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		_ = c.Run(ctx)
	}()
	cancel()
	time.Sleep(50 * time.Millisecond)
}
