import (
	"vela/kube"
)

patchObject: kube.#Patch & {
	$params: {
		resource: {
			apiVersion: parameter.resource.apiVersion
			kind:       parameter.resource.kind
			metadata: {
				name:      parameter.resource.name
				namespace: parameter.resource.namespace
			}
		}
		patch: parameter.patch
	}
}

parameter: {
	// +usage=The resource to patch
	resource: {
		// +usage=The api version of the resource
		apiVersion: string
		// +usage=The kind of the resource
		kind: string
		// +usage=The metadata of the resource
		metadata: {
			// +usage=The name of the resource
			name: string
			// +usage=The namespace of the resource
			namespace: *"default" | string
		}
	}
	// +usage=The patch to be applied to the resource with kubernetes patch
	patch: *{
		// +usage=The type of patch being provided
		type: "merge"
		data: {...}
	} | {
		// +usage=The type of patch being provided
		type: "json"
		data: [{...}]
	} | {
		// +usage=The type of patch being provided
		type: "strategic"
		data: {...}
	}
}
