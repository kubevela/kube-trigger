// This is a validator for properties of patch-k8s-objects

patchTarget: {
	apiVersion: string
	kind:       string
	namespace:  *"" | string
	name:       *"" | string
	labelSelectors?: [string]: string
}

patch: string

allowConcurrency: *false | bool
