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
				// Event type: update, create, delete, leave empty to listen to all events
				events: ["update"]
			}
		}
		// Filter the events above.
		// You can add multiple filters.
		filters: [
			{
				// Filter by validating the object data using CUE.
				// This is a builtin one (and the only one currently).
				type: "cue-validator"
				properties: template: """
					// Filter by object name.
					// I used regular expressions here.
					metadata: name: =~"this-will-trigger-update-.*"
				"""
			},
		]
		// What to do when the events above happen?
		// You can add multiple actions.
		actions: [
			{
				// Bump Application Revision
				type: "bump-application-revision"
				properties: {
					namespace: "default"
					name:      ""
					labelSelectors: {
						"my-label": "my-value"
					}
				}
			},
		]
	},
]
