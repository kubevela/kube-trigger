Described in issue https://github.com/kubevela/kubevela/issues/4418 , sometimes we want to trigger Application update.

This can already do that. Let's see an example.

In `sample.yaml`, we have:

- two ConfigMaps
- two Applications

Although the Applications are empty, let's pretend they have used some ConfigMaps inside. It doesn't matter.

### What we want to achieve?

- Once the two ConfigMaps are updated, the Applications are updated as well.

### Try out

Apply `sample.yaml`

`kubectl apply -f examples/sample.yaml`

Run kube-trigger (will automatically load `examples/sampleconf.cue`. Of course, we will support loading with cli flags.
It is just full of testing code right now).

Edit the two ConfigMaps.

You should see the two Application all have updated: `app.oam.dev/publishVersion: 'xxx'`