# Contributing to kube-trigger

Thanks for helping. This page covers what you need to build, test and send a change. kube-trigger follows KubeVela's
[code of conduct](CODE_OF_CONDUCT.md).

## Finding something to work on

Bugs and planned work are in the [issues](https://github.com/kubevela/kube-trigger/issues). For anything larger than a
small fix, open or comment on an issue first, so the approach can be agreed before you write it.

## What you need

- Go, at the version in [go.mod](go.mod).
- `make`.
- A kubeconfig file, for the tests. They do not talk to a cluster, but some packages load a kubeconfig when they start
  and fail without one.
- Docker, only to build the container images.

## Where things are

| Path | What is there |
| --- | --- |
| `cmd/kubetrigger`, `pkg/` | The worker: sources, filters, the executor that runs actions. |
| `cmd/manager`, `controllers/` | The manager, which turns each TriggerService into a worker. |
| `api/` | The `TriggerService` and `EventListener` types. |
| `config/crd`, `config/manager` | The CRDs, and the manager's manifests. |
| `config/definition` | The built-in actions and the worker template. |
| `examples/` | Examples, with a [guide](examples/README.md). |

## Building

```shell
make build
```

This puts the worker at `bin/kube-trigger` and the manager at `bin/manager`. More build targets, including the container
images, are described in [docs/img/BUILDING.md](docs/img/BUILDING.md) and listed by `make all-help`.

## Testing

```shell
make test
```

A bug fix should come with a test that fails without the fix. Codecov checks that at least 60% of the lines a pull
request changes are covered.

The tests do not need a cluster. To try a change against one, build the binary and run it with a config file as in the
[examples](examples/README.md); it uses the cluster in your kubeconfig.

## Before you send a pull request

```shell
make reviewable
```

This regenerates code, checks the licence headers, runs the linter and tidies `go.mod`. The linter is the pinned version
of golangci-lint, which `make` downloads into `bin/` if yours is a different one. Commit anything it changes. CI runs
the same checks, plus staticcheck.

New Go files need the licence header that the other files start with; `make checklicense` checks for it.

## Pull requests

- **Title.** Start it with what kind of change it is: `Fix: `, `Feat: `, `Docs: `, `Test: `, `Chore: `, `CI: `,
  `Perf: `, `Refactor: `, `Revert: ` or `Style: `. A check enforces this.
- **Sign-off.** Sign every commit with `git commit -s`. The DCO check fails without it.
- **Issue.** Say which issue the change fixes, with `Fixes #123` in the description.
- **One change.** Keep a pull request to one fix or one feature. Pull requests are squashed into a single commit when
  merged, so the title becomes the commit message.
- **Review comments.** Reply to each one, with what you changed or why you did not, and resolve it.

## Adding an action

An action is a CUE template in a `Definition`, so adding one needs no Go. The README shows
[how to write one](README.md#writing-your-own-action). To propose a built-in action, add its Definition under
`config/definition`, and a test that renders it.
