//go:build !ignore_autogenerated

/*
Copyright  The KubeVela Authors.

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

// Code generated by ../../../../hack/generate-go-const-from-file.sh. DO NOT EDIT.

// Instead, edit properties.cue and regenerate this using go generate ./...

package patchk8sobjects

const propertiesCUETemplate = `//+type=patch-k8s-objects
//+description=TODO

//+usage=Select object to patch.
patchTarget: {
	//+usage=Object APIVersion
	apiVersion: string
	//+usage=Object kind
	kind: string
	//+usage=Object namespace. Leave empty to select all namespaces.
	namespace: *"" | string
	//+usage=Object name.
	name: *"" | string
	//+usage=Only path object with these labels.
	labelSelectors?: [string]: string
}
// TODO(charlie0129): parse this multi-line usage
//+usage=Patch is a CUE string that will patch the patchTargets.
//You have some contexts(variables) that you can use in your code:
//  context.event: event metadata
//  context.data: full event data
//  context.target: one of the patchTargets (k8s object) that you selected
//Put the patch in 'output' field, which will be merged with each patchTarget.
patch: string

//+usage=Allow this Action to be run concurrently.
allowConcurrency: *false | bool
`
