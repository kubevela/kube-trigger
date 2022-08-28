// Add as many watchers as you want.
// We add 1 watcher as an example.
triggers: [
	{
		// Watch what event?
		source: {
			// Watch Kubernetes objects.
			type: "k8s-resource-watcher"
			properties: {
				// Watch ConfigMap events.
				apiVersion: "v1"
				kind:       "ConfigMap"
				namespace:  "default"
				// Event type: update, create, delete, leave empty to listen to all events
				events: ["update"]
			}
		}
		// Filter the events above.
		// You can add multiple filters, logical AND will be used.
		filters: [
			{
				// Filter by validating the object data using CUE.
				// For example, we are filtering by ConfigMap names (metadata.name) from above.
				type: "cue-validator"
				properties: template: """
					metadata: name: =~"this-will-trigger-update-.*"
				"""
			},
		]
		// What to do when the events above happen?
		// You can add multiple actions to trigger them all.
		actions: [
			{
				// Bump Application Revision to update Application.
				type: "bump-application-revision"
				properties: {
					namespace: "default"
					// Select Applications.
					labelSelectors: {
						"my-label": "my-value"
					}
				}
			},
		]
	},
]
