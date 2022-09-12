// This is a validator for properties of k8s-resource-watcher

#eventType: "update" | "create" | "delete"

apiVersion: string
kind:       string
namespace:  *"" | string
events: *[] | [...#eventType]
