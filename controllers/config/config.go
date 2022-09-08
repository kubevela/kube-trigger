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

package config

// Config contains options for controllers.
type Config struct {
	// CreateDefaultInstance will make the controller always create a default TriggerInstance.
	CreateDefaultInstance bool
	// ServiceUseDefaultInstance will make TriggerService use the default TriggerInstance if spec.selector is empty.
	ServiceUseDefaultInstance bool
}
