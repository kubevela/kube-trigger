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
	"fmt"
	"strings"
)

// Config is the config for CronJob.
type Config struct {
	Schedule string `json:"schedule"`
	TimeZone string `json:"timeZone"`
}

func (c *Config) String() string {
	if strings.Contains(c.Schedule, "TZ") {
		return c.Schedule
	}

	if c.TimeZone != "" {
		// We don't check if the timezone is valid here.
		return fmt.Sprintf("TZ=%s %s", c.TimeZone, c.Schedule)
	}

	return c.Schedule
}

func formatSchedule(c Config) string {
	if strings.Contains(c.Schedule, "TZ") {
		logger.Warnf("using TZ in schedule is not supported, use timeZone instead")
	}

	return c.String()
}
