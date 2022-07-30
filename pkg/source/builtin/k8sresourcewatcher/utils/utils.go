package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetObjectMetaData(obj interface{}) metav1.Object {
	return obj.(metav1.Object)
}
