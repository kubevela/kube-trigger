Described in issue https://github.com/kubevela/kubevela/issues/4418 , sometimes we want to trigger Application update.

This can already do that. Let's see an example.

In `sample.yaml`, we have:

- two ConfigMaps
- two Applications

### What we want to achieve?

- Once any of the two ConfigMaps are updated, both Applications will be updated as well.

### Try out

- Apply `sample.yaml`
- Run kube-trigger (will automatically load `examples/sampleconf.cue`. Of course, we will support loading with CLI
  flags. It is just full of testing code right now).
- Edit any of the two ConfigMaps.
- You should see the two Application all have updated: `app.oam.dev/publishVersion: '2/3/4...'`

Please read `sampleconf.cue` for details.