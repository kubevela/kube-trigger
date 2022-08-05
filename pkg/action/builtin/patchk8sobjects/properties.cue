// This is a validator for properties of patch-k8s-objects

patchTarget: {
	apiVersion: string
	kind:       string
	namespace:  *"" | string
	name:       *"" | string
	labelSelectors?: [string]: string
}

// patch is a CUE string that will patch the patchTargets.
// Available contexts:
// - context.event: event meta
// - context.data: event data
// - context.target: patchTarget
// Put the patch in 'output' field, which will be merged with each target.
patch: string

allowConcurrency: *false | bool
