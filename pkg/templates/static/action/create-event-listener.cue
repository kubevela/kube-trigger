import (
	"vela/kube"
)

apply: kube.#Apply & {
	$params: {
		resource: {
			apiVersion: "standard.oam.dev/v1alpha1"
			kind:       "EventListener"
			metadata: {
				name:      context.data.metadata.name
				namespace: context.data.metadata.namespace
				if context.data.metadata.labels != _|_ {
					labels: context.data.metadata.labels
				}
				ownerReferences: [
					{
						apiVersion: context.data.apiVersion
						kind:       context.data.kind
						name:       context.data.metadata.name
						uid:        context.data.metadata.uid
						controller: true
					},
				]
			}
		}
	}
}
