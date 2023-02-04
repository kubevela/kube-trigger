import (
	"vela/kube"
	"strconv"
)

list: kube.#List & {
	$params: {
		resource: {
			apiVersion: "core.oam.dev/v1beta1"
			kind:       "Application"
		}
		filter: {
			namespace: parameter.namespace
			if parameter.matchingLabels != _|_ {
				matchingLabels: parameter.matchingLabels
			}
		}
	}
}

for index, item in list.$returns.items {
	if item.metadata.annotations["app.oam.dev/publishVersion"] != _|_ {
		if strconv.ParseInt(item.metadata.annotations["app.oam.dev/publishVersion"], 10, 64) != _|_ {
			"patch-\(index)": kube.#Patch & {
				$params: {
					resource: {
						apiVersion: "core.oam.dev/v1beta1"
						kind:       "Application"
						metadata: {
							name:      item.metadata.name
							namespace: item.metadata.namespace
						}
					}
					patch: {
						type: "merge"
						data: {
							metadata: {
								annotations: {
									"app.oam.dev/publishVersion": strconv.FormatInt(strconv.ParseInt(item.metadata.annotations["app.oam.dev/publishVersion"], 10, 64)+1, 10)
								}
							}
						}
					}
				}
			}
		}
	}
}

parameter: {
	// +usage=The namespace to list the resources
	namespace?: *"" | string
	// +usage=The label selector to filter the resources
	matchingLabels?: {...}
}
