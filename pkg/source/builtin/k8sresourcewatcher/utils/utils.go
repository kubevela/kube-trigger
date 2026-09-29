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

package utils

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utiljson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/cache"
)

// GetObjectMetaData returns the object an informer event is about, unwrapping
// the tombstone of a delete seen only on a relist. It reports false when there
// is no object, which a tombstone can also carry.
func GetObjectMetaData(obj interface{}) (metav1.Object, bool) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	o, ok := obj.(metav1.Object)
	return o, ok
}

// MergePatch returns the RFC 7386 merge patch that turns from into to: the
// fields that differ with their new values, a removed field as null, and a
// changed list whole. It is decoded as unstructured objects are, so whole
// numbers stay int64 for CUE.
func MergePatch(from, to interface{}) (map[string]interface{}, error) {
	fromJSON, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}
	toJSON, err := json.Marshal(to)
	if err != nil {
		return nil, err
	}
	patchJSON, err := jsonpatch.CreateMergePatch(fromJSON, toJSON)
	if err != nil {
		return nil, err
	}
	patch := map[string]interface{}{}
	return patch, utiljson.Unmarshal(patchJSON, &patch)
}
