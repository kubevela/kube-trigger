//+type=patch-k8s-objects
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
//+usage=Patch is a CUE string that will patch the patchTargets. \n\
//You have some contexts(variables) that you can use in your code: \n\
//  context.event: event metadata \n\
//  context.data: full event data \n\
//  context.target: one of the patchTargets (k8s object) that you selected \n\
//Put the patch in 'output' field, which will be merged with each patchTarget. \n\
patch: string

//+usage=Allow this Action to be run concurrently.
allowConcurrency: *false | bool
