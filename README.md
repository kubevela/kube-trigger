# kube-trigger

> This project is in its early stage as a proof-of-concept. Don't expect to be able to use it in production any time
> soon.
>
> **You can check out `examples/` to see what it can do now.** The overall architectural design is almost done.

kube-trigger can list and watch kubernetes object events and run actions. The project is inspired
by [kubewatch](https://github.com/vmware-archive/kubewatch).

It can solve issues like https://github.com/kubevela/kubevela/issues/4418 .

It's a lightweight event-trigger in Kubernetes world. The
architecture can be:

```                                                                                         
       Kubernetes Events                                             Operations on Kubernetes
       Cron by time                     Conditions                   or any API Notifications
                                                                                             
    +--------------------+          +---------------------+          +---------------------+ 
    |                    |          |                     |          |                     | 
    |      Sources       ----------->       Filters       ----------->       Actions       | 
    |                    |          |                     |          |                     | 
    +--------------------+          +---------------------+          +---------------------+ 
                                                                                            
                                                                                             
    +--------------------+          +---------------------+          +---------------------+ 
    |                    |          |                     |          |                     | 
    |      Sources       ----------->       Filters       ----------->       Actions       | 
    |                    |          |                     |          |                     | 
    +--------------------+          +---------------------+          +---------------------+ 
```

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
- [ ] Make the configuration as CRD, launch new process/pod for new watcher
