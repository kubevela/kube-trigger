# Examples

The examples here were run against a cluster before they were written down, on main's code, and what each needs and
what you should see are listed with it. The one exception is the example under [Not run](#not-run).

| File | Shows | Needs |
| --- | --- | --- |
| [triggerservice-audit-configmap.yaml](triggerservice-audit-configmap.yaml), [conf-audit-configmap.yaml](conf-audit-configmap.yaml) | The basics: watch a resource, filter, patch another resource. | |
| [triggerservice-cel.yaml](triggerservice-cel.yaml) | A CEL filter, and CEL expressions in action properties. | |
| [triggerservice-on-change.yaml](triggerservice-on-change.yaml) | Acting only when one part of an object changes. | |
| [conf-cronjob.yaml](conf-cronjob.yaml) | Running an action on a schedule. | |
| [triggerservice-bump-app.yaml](triggerservice-bump-app.yaml), [conf-bump-app.yaml](conf-bump-app.yaml), [sample.yaml](sample.yaml) | Redeploying a KubeVela Application when a ConfigMap changes. | KubeVela |
| [triggerservice-record-event.yaml](triggerservice-record-event.yaml), [conf-create-event-listener.yaml](conf-create-event-listener.yaml), [conf-record-event.yaml](conf-record-event.yaml) | Recording events in an `EventListener`. | The WorkflowRun CRD |

`triggerservice-*.yaml` files are applied to a cluster. `conf-*.yaml` files hold the same triggers for a standalone
kube-trigger.

## Before you start

Everything here is in the `default` namespace. If that is not your current namespace, add `-n default` to the
`kubectl` commands.

Install kube-trigger from the root of this repository:

```shell
kubectl apply -f config/crd/
kubectl apply -f config/manager/
kubectl apply -f config/definition/
```

Most examples use two ConfigMaps, one to change and one to write to:

```shell
kubectl create configmap app-settings --from-literal=color=blue
kubectl create configmap settings-audit
```

To run a `conf-*.yaml` file instead of a TriggerService, build the binary (see [CONTRIBUTING.md](../CONTRIBUTING.md)) and
point it at the file. It uses the cluster in your kubeconfig:

```shell
bin/kube-trigger --config examples/conf-audit-configmap.yaml
```

A standalone kube-trigger does not need the manager. If you only run `conf-*.yaml` files, create the namespace
yourself in place of applying `config/manager/`:

```shell
kubectl apply -f config/crd/
kubectl create namespace vela-system
kubectl apply -f config/definition/
```

After applying a TriggerService, the manager takes a few seconds to create its worker Deployment. If
`kubectl rollout status` says the Deployment is not found, run it again.

## Watch, filter, act

Records when `app-settings` was last changed.

```shell
kubectl apply -f examples/triggerservice-audit-configmap.yaml
kubectl rollout status deployment/audit-settings-changes

kubectl patch configmap app-settings --type merge -p '{"data":{"color":"green"}}'
kubectl get configmap settings-audit -o jsonpath='{.data}'
```

```
{"lastChanged":"2026-11-27T22:16:18Z"}
```

Updating any other ConfigMap leaves `settings-audit` as it is.

## CEL

Records the last create or update of any ConfigMap labelled `audit=true`. The filter is CEL, and guards the label with
`has()` and `in`, because reading a label an object does not have is an error in CEL. The action's properties build
their values from the event with `$( )`.

```shell
kubectl apply -f examples/triggerservice-cel.yaml
kubectl rollout status deployment/audit-labelled-configmaps

kubectl create configmap web-config --from-literal=a=1
kubectl label configmap web-config audit=true
kubectl get configmap settings-audit -o jsonpath='{.data}'
```

```
{"lastEvent":"update","lastObject":"web-config","summary":"update of web-config at 2026-11-27T22:16:33Z"}
```

Creating `web-config` without the label changed nothing; adding the label is the update that was recorded.

## Acting on what changed

Records a change to the data of `app-settings`, and ignores changes to its labels or annotations, by testing
`context.changed`.

```shell
kubectl apply -f examples/triggerservice-on-change.yaml
kubectl rollout status deployment/audit-settings-data

kubectl label configmap app-settings touched=yes
kubectl get configmap settings-audit -o jsonpath='{.data}'    # no dataLastChanged yet

kubectl patch configmap app-settings --type merge -p '{"data":{"color":"orange"}}'
kubectl get configmap settings-audit -o jsonpath='{.data}'
```

```
{"dataLastChanged":"2026-11-27T22:16:49Z"}
```

## On a schedule

Writes the time into `settings-audit` every minute. This one is a config file, run standalone:

```shell
bin/kube-trigger --config examples/conf-cronjob.yaml
```

Within a minute:

```shell
kubectl get configmap settings-audit -o jsonpath='{.data}'
```

```
{"lastTick":"2026-11-27T22:18:00Z"}
```

## Redeploying a KubeVela Application

Needs [KubeVela](https://kubevela.io) in the cluster; this was run with KubeVela 1.11.2, as a config file with a
standalone kube-trigger. `triggerservice-bump-app.yaml` holds the same trigger for use in a cluster.

The use case is [kubevela#4418](https://github.com/kubevela/kubevela/issues/4418): an Application should be deployed
again when a ConfigMap it uses changes. `sample.yaml` creates two ConfigMaps and two Applications. Each ConfigMap names
its Application in a `watch-this` label, and the trigger bumps that Application's `app.oam.dev/publishVersion`.

```shell
kubectl apply -f examples/sample.yaml
bin/kube-trigger --config examples/conf-bump-app.yaml
```

In another terminal:

```shell
kubectl patch configmap this-will-trigger-update-1 --type merge -p '{"data":{"content":"edited"}}'
kubectl get applicationrevision
```

```
NAME                        AGE   PUBLISH_VERSION   SUCCEEDED
this-will-be-updated-1-v1   33s   1                 true
this-will-be-updated-1-v2   15s   2                 true
this-will-be-updated-2-v1   33s   1                 true
```

`this-will-be-updated-1` got a new revision. `this-will-be-updated-2` did not, as its ConfigMap was not edited.

## Recording events

`triggerservice-record-event.yaml` has two triggers:

1. When a WorkflowRun labelled `trigger.oam.dev/watch=true` is created, `create-event-listener` creates an
   `EventListener` with its name, owned by it.
2. When a Deployment labelled `trigger.oam.dev/watch=true` becomes fully ready, `record-event` appends the event to the
   `EventListener` named by the Deployment's `workflowrun.oam.dev/name` label.

It needs the WorkflowRun CRD from [kubevela/workflow](https://github.com/kubevela/workflow).

While the Deployment has no ready replica yet, the worker logs `filter: undefined field: readyReplicas` for each of its
updates. Those events are skipped, and the one where it becomes ready is recorded.

## Cleaning up

Deleting a TriggerService removes its worker. The rest is what the walkthroughs above created; skip the lines for those
you did not run.

```shell
kubectl delete triggerservice --all
kubectl delete configmap app-settings settings-audit
kubectl delete configmap web-config          # CEL
kubectl delete -f examples/sample.yaml       # Redeploying a KubeVela Application
```

## Not run

[triggerservice-image-update.yaml](triggerservice-image-update.yaml) watches kpack `Image` resources and needs kpack
installed. It has not been run as part of checking these examples.
