import (
	"vela/kube"
	"strconv"
)

meta: {
	apiVersion: "standard.oam.dev/v1alpha1"
	kind:       "EventListener"
	metadata: {
		name:      context.data.metadata.name
		namespace: context.data.metadata.namespace
	}
}

get: kube.#Get & {
	$params: {
		resource: meta
	}
}

originalEvents: *[] | [...]

if get.$returns.events != _|_ {
	originalEvents: get.$returns.events
}

events: originalEvents + [{
	resource: {
		apiVersion: context.data.apiVersion
		kind:       context.data.kind
		name:       context.data.metadata.name
		namespace:  context.data.metadata.namespace
	}
	type:      context.event.type
	timestamp: context.timestamp
}]

filter: *events | [...]

if len(events) > 10 {
	filter: events[len(events)-10:]
}

"patch": kube.#Patch & {
	$params: {
		resource: meta
		patch: {
			type: "merge"
			data: {
				events: filter
			}
		}
	}
}
