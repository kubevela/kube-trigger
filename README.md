# kube-trigger

kube-trigger is a tool that combines event listeners and action triggers.

![kube-trigger overview](docs/img/overview.svg)

Although there is `kube` in the name, it is actually not limited to Kubernetes and can do much more than that. It has an
extensible architecture that can extend its capabilities fairly easily.

We provide generic low-level filters/actions to users, and users can create customized filters/actions to wrap low-level
ones using CUE.

## TODO:

- [x] Basic build infrastructure
- [x] Complete a basic proof-of-concept sample
- [x] linters, license checker
- [ ] GitHub Actions
- [ ] **Add tests. No tests currently, this is terrible.**
- [ ] Make it run as Addon, build component definition, and examples
- [x] Notification for more than one app: selector from compose of Namespace; Labels; Name
- [ ] Refine README, quick starts, contribution guide
- [ ] More Source types
- [ ] More Action Types, such as WorkflowRun, API webhook, notifications(email, dingtalk, slack), execute velaql(
  CUE and K8s operations), metric (prometheus), storage (clickhouse)
- [ ] Allow users to extend builtin filters/actions
- [ ] Remove cache informer, make it with no catch but list watch events with unique queue.
- [x] Make the configuration as CRD, launch new process/pod for new watcher
