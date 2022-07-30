// Watch what event?
watch: [
	{
		source: {
			// Kubernetes object info
			// Currnetly, this can only be used once.
			type: "k8s-resource-watcher"
			properties: {
				apiVersion: "v1"
				kind:       "ConfigMap"
				namespace:  "default"
				// Event type: update, create, delete, or ""
				// Currently unsupported.
				// event: "update"
			}
		}
		// Filter the events above
		filters: [
			{
				// Check the object data using cue
				type: "cue-validator"
				properties: template: {
					// Filter by object name
					metadata: name: "my-cm-1"
				}
			},
		]
	},
]

// What to do when the events above happen?
actions: [
	{
		// Update Kubernetes objects
		type: "update-k8s-object"
		properties: {
			patchTarget: {
				apiVersion: "v1"
				kind:       "ConfigMap"
				namespace:  "default"
				name:       "my-cm-2"
			}
			// context.sourceObject, context.patchTarget is filled automatically
			// output is deep merged with context.obj
			patch: {
				data: somecontent: context.sourceObject.data.somecontent
			}

			allowConcurrent: false
		}
	},
	// {
	//  // Execute a command
	//  type: "execute"
	//  properites: {
	//   path: "bash"
	//   args: ["-c"]
	//   timeout: "10s"
	//  }
	// },
]
