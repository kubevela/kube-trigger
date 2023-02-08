# kube-trigger

[![Go Report Card](https://goreportcard.com/badge/github.com/kubevela/kube-trigger)](https://goreportcard.com/report/github.com/kubevela/kube-trigger)
[![LICENSE](https://img.shields.io/github/license/kubevela/kube-trigger.svg?style=flat-square)](/LICENSE)
[![Releases](https://img.shields.io/github/release/kubevela/kube-trigger/all.svg?style=flat-square)](https://github.com/kubevela/kube-trigger/releases)

kube-trigger is a tool that combines event listeners and action triggers.

![kube-trigger overview](docs/img/overview.svg)

## Overview

Although there is `kube` in the name, it is actually not limited to Kubernetes and can do much more than that. It has an
extensible architecture that can extend its capabilities fairly easily. We have docs (not yet) on how to
extend [Source](#Source), [Filter](#Filter), and [Action](#Action). All users are welcomed to contribute their own
extensions.

### Source

A Source is what listens to events (event source). For example, a `resource-watcher` source can watch Kubernetes
resources from multi-clusters. Once a Kubernetes resource (e.g. Deployment) is changed, it will raise an event that will be passed
to [Filter](#Filter) for further processing.

```yaml
source:
  type: resource-watcher
  properties:
    apiVersion: apps/v1
    kind: Deployment
    events:
      - update
```

### Filters

A Filter will filter the events that are raised by [Source](#Source), i.e, drop events that do not satisfy a certain
criteria. For example, users can check the status of the Deployment to decide whether to filter out events. All the
events that passed the Filter will then trigger an [Action](#Action).

```yaml
filter: context.data.status.readyReplicas == context.data.status.replicas
```

### Actions

An Action is a job that does what the user specified when an event happens. For example, the user can send
notifications, log events, execute a command, or patch some Kubernetes objects when an event happens.

For example, you can use the built-in action `patch-resource` to patch resources in Kubernetes like:

```yaml
action:
  type: patch-resource
  properties:
    resource:
      apiVersion: v1
      kind: ConfigMap
      name: my-cm
      namespace: default
    patch:
      type: merge
      data:
        data: 
          foo: bar
```

The underlying mechanism of action is using [CUE](https://github.com/cue-lang/cue) to render the template and execute it with your parameters. For example, the above action can be written in CUE as:

```cue
import (
	"vela/kube"
)

patchObject: kube.#Patch & {
	$params: {
		resource: {
			apiVersion: parameter.resource.apiVersion
			kind:       parameter.resource.kind
			metadata: {
				name:      parameter.resource.name
				namespace: parameter.resource.namespace
			}
		}
		patch: parameter.patch
	}
}

// users' parameters to be passed to the action
parameter: {
	// +usage=The resource to patch
	resource: {
		// +usage=The api version of the resource
		apiVersion: string
		// +usage=The kind of the resource
		kind: string
		// +usage=The metadata of the resource
		metadata: {
			// +usage=The name of the resource
			name: string
			// +usage=The namespace of the resource
			namespace: *"default" | string
		}
	}
	// +usage=The patch to be applied to the resource with kubernetes patch
	patch: *{
		// +usage=The type of patch being provided
		type: "merge"
		data: {...}
	} | {
		// +usage=The type of patch being provided
		type: "json"
		data: [{...}]
	} | {
		// +usage=The type of patch being provided
		type: "strategic"
		data: {...}
	}
}

```

## Quick Start

To quickly know the concepts of kube-trigger, let's use a real use-case as an exmaple (
see [#4418](https://github.com/kubevela/kubevela/issues/4418)). TL;DR, the user want the Application to be automatically
updated whenever the ConfigMaps that are referenced by `ref-objects` are updated.

To accomplish this, we will:

- use a `resource-watcher` Source to listen to update events of ConfigMaps
- filter the events to only keep the ConfigMaps that we are interested in
- trigger an `bump-application-revision` Action to update Application.

And the trigger config file will look like:

```yaml
triggers:
  - source:
      type: resource-watcher
      properties:
        # We are interested in ConfigMap events.
        apiVersion: "v1"
        kind: ConfigMap
        namespace: default
        # Only watch update event.
        events:
          - update
    filter: |
      context: data: metadata: name: =~"this-will-trigger-update-.*"
    action:
      # Bump Application Revision to update Application.
      type: bump-application-revision
      properties:
        namespace: default
        # Select Applications to bump using labels.
        matchingLabels:
          my-label: my-value
```

See [examples](https://github.com/kubevela/kube-trigger/tree/main/examples) directory for more instructions.

## Usage

You can run kube-trigger in two modes: standalone and in-cluster.

### Configuration File

A config file instructs kube-trigger to use what [Source](#Source), [Filter](#Filter), and [Action](#Action), and
how they are configured.

No matter you are running kube-trigger as standalone or in-cluster, the config format is similar, so it is beneficial to
know the format first. We will use yaml format as an example (json and cue are also supported).

```yaml
# A trigger is a group of Source, Filter, and Action.
# You can add multiple triggers.
triggers:
  - source:
      type: <your-source-type>
      properties: ...
      # ... properties
    filter: <your-filter>
    action:
      type: <your-action-type>
      properties: ...
```

### Standalone

When running in standalone mode, you will need to provide a config file to kube-trigger binary.

kube-trigger can accept `cue`, `yaml`, and `json` config files. You can also specify a directory to load all the
supported files inside that directory. `-c`/`--config` cli flag and `CONFIG` environment variable can be used to specify
config file.

An example config file looks like this:

```yaml
# A trigger is a group of Source, Filters, and Actions.
# You can add multiple triggers.
triggers:
  - source:
      type: resource-watcher
      properties:
        # We are interested in ConfigMap events.
        apiVersion: "v1"
        kind: ConfigMap
        namespace: default
        # Only watch update event.
        events:
          - update
    # Filter the events above.
    filter: |
      context: data: metadata: name: =~"this-will-trigger-update-.*"
    action:
      # Bump Application Revision to update Application.
      type: bump-application-revision
      properties:
        namespace: default
        # Select Applications to bump using labels.
        matchingLabels:
          my-label: my-value
```

Let's assume your config file is `config.yaml`, to run kube-trigger:

- `./kube-trigger -c=config.yaml`
- `CONFIG=config.yaml ./kube-trigger`

### In-Cluster

We have one CRD called *TriggerService*. *TriggerInstance* is what creates a kube-trigger instance (similar to running `./kube-trigger` in-cluster but no config with config specified in its spec.

```yaml
# You can find this file in config/samples/standard_v1alpha1_triggerservice.yaml
apiVersion: standard.oam.dev/v1alpha1
kind: TriggerService
metadata:
  name: kubetrigger-sample-config
  namespace: default
spec:
  selector:
    instance: kubetrigger-sample
  triggers:
    - source:
        type: resource-watcher
        properties:
          apiVersion: "v1"
          kind: ConfigMap
          namespace: default
          events:
            - update
      filter: |
        context: data: metadata: name: =~"this-will-trigger-update-.*"
      action:
        type: bump-application-revision
        properties:
          namespace: default
          matchingLabels:
            my-label: my-value
```

## Advanced kube-trigger Configuration

In addition to config files, you can also do advanced configurations. Advanced kube-trigger Configurations are
internal configurations to fine-tune your kube-trigger instance. In
most cases, you probably don't need to fiddle with these settings.

### Log Level

Frequently-used values: `debug`, `info`, `error`

Default: `info`

| CLI           | ENV         | KubeTrigger CRD |
|---------------|-------------|-----------------|
| `--log-level` | `LOG_LEVEL` | `TODO`          |

### Action Retry

Re-run Action if it fails.

Default: `false`

| CLI              | ENV            | KubeTrigger CRD |
|------------------|----------------|-----------------|
| `--action-retry` | `ACTION_RETRY` | `TODO`          |

### Max Retry

Max retry count if an Action fails, valid only when action retrying is enabled.

Default: `5`

| CLI           | ENV         | KubeTrigger CRD               |
|---------------|-------------|-------------------------------|
| `--max-retry` | `MAX_RETRY` | `.spec.workerConfig.maxRetry` |

### Retry Delay

First delay to retry actions in seconds, subsequent delay will grow exponentially, valid only when action retrying is
enabled.

Default: `2`

| CLI             | ENV           | KubeTrigger CRD                 |
|-----------------|---------------|---------------------------------|
| `--retry-delay` | `RETRY_DELAY` | `.spec.workerConfig.retryDelay` |

### Per-Worker QPS

Long-term QPS limiting per Action worker, this is shared between all watchers.

Default: `2`

| CLI                | ENV              | KubeTrigger CRD                   |
|--------------------|------------------|-----------------------------------|
| `--per-worker-qps` | `PER_WORKER_QPS` | `.spec.workerConfig.perWorkerQPS` |

### Queue Size

Queue size for running actions, this is shared between all watchers

Default: `50`

| CLI            | ENV          | KubeTrigger CRD                |
|----------------|--------------|--------------------------------|
| `--queue-size` | `QUEUE_SIZE` | `.spec.workerConfig.queueSize` |

### Job Timeout

Timeout for running each action in seconds.

Default: `10`

| CLI         | ENV       | KubeTrigger CRD              |
|-------------|-----------|------------------------------|
| `--timeout` | `TIMEOUT` | `.spec.workerConfig.timeout` |

### Worker Count

Number of workers for running actions, this is shared between all watchers.

Default: `4`

| CLI         | ENV       | KubeTrigger CRD                  |
|-------------|-----------|----------------------------------|
| `--workers` | `WORKERS` | `.spec.workerConfig.workerCount` |

### Registry Size

Cache size for filters and actions.

Default: `100`

| CLI               | ENV             | KubeTrigger CRD      |
|-------------------|-----------------|----------------------|
| `--registry-size` | `REGISTRY_SIZE` | `.spec.registrySize` |

## Roadmap

### v0.0.1-alpha.x

- [x] Basic build infrastructure
- [x] Complete a basic proof-of-concept sample
- [x] linters, license checker
- [x] GitHub Actions
- [x] Rate-limited worker
- [x] Make the configuration as CRD, launch new process/pod for new watcher
- [x] Notification for more than one app: selector from compose of Namespace; Labels; Name
- [x] Refine README, quick starts
- [x] Refactor CRD according to [#2](https://github.com/kubevela/kube-trigger/issues/2)

### v0.0.1-beta.x

Code enhancements

- [ ] Add missing unit tests
- [ ] Add missing integration tests

### v0.0.x

User experience

- [ ] Refine health status of CRs
- [ ] Make it run as Addon, build component definition, and examples
- [ ] Kubernetes dynamic admission control with validation webhook
- [ ] Auto-generate usage docs of Sources, Filters, and Actions from CUE markers
- [ ] Show available Sources, Filters, and Actions in cli

### v0.1.x

Webhook support

- [ ] Contribution Guide
- [ ] New Action: webhook
- [ ] New Source: webhook

### v0.2.x

Observability

- [ ] New Action: execute VelaQL(CUE and K8s operations)
- [ ] New Source: cron
- [ ] New Action: notifications(email, dingtalk, slack, telegram)
- [ ] New Action: log (loki, clickhouse)

### Planned for later releases

- [ ] Allow user set custom RBAC for each TriggerInstance
- [ ] New Action: workflow-run
- [ ] New Action: execute-command
- [ ] New Action: metric (prometheus)
- [ ] Refine controller logic
- [ ] Remove cache informer, make it with no catch but list watch events with unique queue.


