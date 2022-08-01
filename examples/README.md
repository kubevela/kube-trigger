Described in issue https://github.com/kubevela/kubevela/issues/4418 , sometimes we want k8s event to trigger an
Application update.

Current kube-trigger can already do that. Let's see an example.

In `sample.yaml`, we have:

- two ConfigMaps
- two Applications

### What we want to achieve?

- Once any of the two ConfigMaps are updated, both Applications will be updated as well.

### Try out

1. **Apply `sample.yaml`**

```shell
kubectl apply examples/sample.yaml
```

2. **Run kube-trigger**

```shell
make dirty-build
bin/kube-trigger --config examples/sampleconf.cue
```

3. **Watch ApplicationRevision changes** so that you can see what it does.

```shell
kubectl get apprev --watch
```

4. **Edit any of the two ConfigMaps** (do this in another terminal)

```shell
kubectl edit cm this-will-trigger-update-1
```

Immediately, you should see the two new ApplicationRevision created.

Specifically, Applications all have updated with annotation: `app.oam.dev/publishVersion: '2/3/4...'`

Please read `sampleconf.cue` for more details.