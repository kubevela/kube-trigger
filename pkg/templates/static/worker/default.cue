// deployment will be renderd and applied to the cluster
deployment: {
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {
		name:      parameter.name
		namespace: triggerService.namespace
		labels: {
			"app.kubernetes.io/name": parameter.name
			"trigger.oam.dev/name":   triggerService.name
		}
	}
	spec: {
		selector: {
			matchLabels: {
				"app.kubernetes.io/name": parameter.name
				"trigger.oam.dev/name":   triggerService.name
			}
		}
		replicas: 1
		template: {
			metadata: {
				labels: {
					"app.kubernetes.io/name": parameter.name
					"trigger.oam.dev/name":   triggerService.name
				}
			}
			spec: {
				securityContext: {
					runAsNonRoot: true
					seccompProfile: {
						type: "RuntimeDefault"
					}
				}
				containers: [{
					workingDir: "/"
					args: [
						"-c=/etc/kube-trigger",
						"--log-level=debug",
						"--max-retry=\(parameter.config.maxRetry)",
						"--retry-delay=\(parameter.config.retryDelay)",
						"--per-worker-qps=\(parameter.config.perWorkerQPS)",
						"--queue-size=\(parameter.config.queueSize)",
						"--timeout=\(parameter.config.timeout)",
						"--workers=\(parameter.config.workers)",
						"--log-level=\(parameter.config.logLevel)",
						"--multi-cluster-config-type=\(parameter.config.multiClusterConfigType)",
					]
					image: parameter.image
					name:  "kube-trigger"
					securityContext: {
						allowPrivilegeEscalation: false
						capabilities: {
							drop: ["ALL"]
						}
					}
					resources: {
						limits: {
							cpu:    parameter.resource.cpu.limits
							memory: parameter.resource.memory.limits
						}
						requests: {
							cpu:    parameter.resource.cpu.requests
							memory: parameter.resource.memory.requests
						}
					}
					volumeMounts: [{
						mountPath: "/etc/kube-trigger"
						name:      "config"
					}]
				}]
				serviceAccountName:            parameter.serviceAccount
				terminationGracePeriodSeconds: 10
				volumes: [{
					name: "config"
					configMap: {
						name: triggerService.name
					}
				}]
			}
		}
	}
}

triggerService: {
	name:      string
	namespace: *"vela-system" | string
}

parameter: {
	name:  *triggerService.name | string
	image: *"oamdev/kube-trigger:latest" | string
	resource: {
		cpu: {
			requests: *"10m" | string
			limits:   *"500m" | string
		}
		memory: {
			requests: *"64Mi" | string
			limits:   *"128Mi" | string
		}
	}
	serviceAccount: *"kube-trigger" | string
	config: {
		maxRetry:               *5 | int
		retryDelay:             *2 | int
		perWorkerQPS:           *2 | int
		queueSize:              *50 | int
		timeout:                *10 | int
		workers:                *4 | int
		logLevel:               *"info" | "debug"
		multiClusterConfigType: *"cluster-gateway" | "cluster-gateway-secret"
	}
}
