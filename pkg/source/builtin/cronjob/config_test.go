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

import "testing"

func TestFormatSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		timeZone string
		want     string
	}{
		{
			name:     "schedule_without_timezone",
			schedule: "* * * * *",
			timeZone: "",
			want:     "* * * * *",
		},
		{
			name:     "schedule_with_timezone",
			schedule: "* * * * *",
			timeZone: "Asia/Shanghai",
			want:     "TZ=Asia/Shanghai * * * * *",
		},
		{
			name:     "schedule_with_timezone_prefixed_unsupported_but_will_work",
			schedule: "TZ=Asia/Shanghai * * * * *",
			timeZone: "",
			want:     "TZ=Asia/Shanghai * * * * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Schedule: tt.schedule,
				TimeZone: tt.timeZone,
			}
			if got := formatSchedule(*c); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
