package builtin

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Filter interface {
	ApplyToObject(obj metav1.Object) (bool, error)
	Type() string
	Init(properties map[string]interface{}) error
	New() Filter
}
