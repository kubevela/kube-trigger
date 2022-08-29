# Quick Start

kube-trigger can run as standalone or in-cluster. Let's use a real use-case as an exmaple (
see [#4418](https://github.com/kubevela/kubevela/issues/4418)). TL;DR, the user want the Application to be automatically
updated whenever the ConfigMaps that are referenced by `ref-objects` are updated.

## Prerequisites

- Install [KubeVela](https://kubevela.net/docs/install) in your cluster

## What we want to achieve?

- use a `k8s-resource-watcher` Source to listen to update events of ConfigMaps
- use a `cue-validator` Filter to only keep the ConfigMaps that we are interested in
- trigger an `bump-application-revision` Action to update Application.

As a result:

- Once any of the two ConfigMaps are updated, both Applications will be updated as well.

## Try out

1. **Apply sample resources**

Apply `sample.yaml` to create 2 Applications and 2 ConfigMaps in the default namespace. The changes in 2 ConfigMaps will
trigger 2 Application updates.

```shell
kubectl apply sample.yaml
```

2. **Run kube-trigger**

Choose your preferred way: standalone (recommended for quick testing) or in-cluster

Standalone:

```shell
# Download kube-trigger binaries from releases first
./kube-trigger --config sampleconf.yaml
```

In-Cluster:

```shell
# Install CRDs: TriggerInstance and TriggerService
# - *TriggerInstance* is what creates a kube-trigger instance (similar to running 
#   `./kube-trigger` in-cluster but no config is provided). 
# - *TriggerService* is used to provide one or more configs (same as the
#   config file you use when running as standalone) to a *TriggerInstance*.
kubectl apply -f config/crd
# Create namespace
kubectl apply -f config/manager/ns.yaml
# Run controllers
kubectl apply -f config/manager
# Create a KubeTrigger instance
kubectl apply -f config/samples/standard_v1alpha1_triggerinstance.yaml
# Add config to the KubeTrigger instance by creating a TriggerService
kubectl apply -f config/samples/standard_v1alpha1_triggerservice.yaml
```

3. **Watch ApplicationRevision changes** so that you can see what it does.

```shell
kubectl get apprev --watch
```

4. **Edit any of the two ConfigMaps** (do this in another terminal)

```shell
kubectl edit cm this-will-trigger-update-1
```

Immediately, you should see the two new ApplicationRevision created. Specifically, Applications all have updated with
annotation: `app.oam.dev/publishVersion: '2/3/4...'`

Please read `sampleconf.cue/yaml` for more details.

## Delete resources

Just replace all `kubectl apply` with `kubectl delete`, and run them in the reverse order.