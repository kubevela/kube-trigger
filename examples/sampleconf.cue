// Add as many watchers as you want.
// We add 1 watcher as an example.
watchers: [
	{
		// Watch what event?
		source: {
			// Watch Kubernets objects.
			// This is a builtin one (and the only one currently).
			type: "k8s-resource-watcher"
			properties: {
				apiVersion: "v1"
				kind:       "ConfigMap"
				namespace:  "default"
				// Event type: update, create, delete, or ""
				event: "update"
			}
		}
		// Filter the events above.
		// You can add multiple filters.
		filters: [
			{
				// Filter by validating the object data using CUE.
				// This is a builtin one (and the only one currently).
				type: "cue-validator"
				properties: template: {
					// Filter by object name.
					// I used regular expressions here.
					metadata: name: =~"this-will-trigger-update-.*"
				}
			},
		]
		// What to do when the events above happen?
		// You can add multiple actions.
		actions: [
			{
				// Patch Kubernetes objects (update a whole list of objects).
				// This is a builtin one (and the only one currently).
				type: "patch-k8s-objects"
				// Use these clues to list objects.
				properties: {
					patchTarget: {
						apiVersion: "core.oam.dev/v1beta1"
						kind:       "Application"
						namespace:  "default"
						name:       ""
						labelSelectors: {
							"my-label": "my-value"
						}
					}
					// Patch will apply the patch to each object and update each one.
					// **For example, we are bumping "app.oam.dev/publishVersion" here.**
					// We have builtin context that you can use:
					// - context.sourceObject (object from "k8s-resource-watcher", will be ConfigMap here)
					// - context.patchTarget (object from patchTarget above, will be Application here)
					patch: """
						import "strconv"
						// This will bump publishVersion by 1
						output: metadata: annotations: {
							"app.oam.dev/publishVersion": strconv.FormatInt(strconv.ParseInt(context.patchTarget.metadata.annotations["app.oam.dev/publishVersion"], 10, 64)+1, 10)
						}
						"""
					// This action will not be running concurrently.
					// For this one, this is disabled by default.
					// For other types of actions, they can run concurrently if you want.
					allowConcurrency: false
				}
			},
		]
	},
]
