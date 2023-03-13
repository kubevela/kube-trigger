triggers: [{
	source: {
		type: "resource-watcher"
		properties: {
			clusters: [
				"cn-shanghai",
			]
			apiVersion: "apps/v1"
			kind:       "Deployment"
			events: [
				"update",
			]
		}
	}
	filter: "context.data.status.readyReplicas == context.data.status.replicas"
	action: {
		type: "sae-record-event"
		properties: nameSelector: fromLabel: "workflowrun.oam.dev/name"
	}
}]
