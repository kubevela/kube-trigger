# kube-trigger

> This project is in its early stage as a proof-of-concept. Don't expect to be able to use it in production any time
> soon.
>
> **You can check out `examples/` to see what it can do now.** The overall architectural design is almost done.

kube-trigger can list and watch kubernetes object event and trigger an event to destination. The project is inspired
by [kubewatch](https://github.com/vmware-archive/kubewatch).

Currently, the basic usage of kube-trigger is to watch any kind of Kubernetes CRD and trigger update of Application. It
can solve issues like https://github.com/kubevela/kubevela/issues/4418 .

But the usage of kube-trigger is more than that, actually it's a lightweight event-trigger in Kubernetes world. The
architecture can be:

```                                                                                         
       Kubernetes Events                                              Operations on Kubernetes or any API
       Cron by time                     Conditions                      Notifications        
                                                                                             
    +--------------------+          +---------------------+          +---------------------+ 
    |                    |          |                     |          |                     | 
    |      Sources       ----------->       Filters       ----------->     Destinations    | 
    |                    |          |                     |          |                     | 
    +--------------------+          +---------------------+          +---------------------+ 
                                                                                            
                                                                                             
    +--------------------+          +---------------------+          +---------------------+ 
    |                    |          |                     |          |                     | 
    |      Sources       ----------->       Filters       ---------->-     Destinations    | 
    |                    |          |                     |          |                     | 
    +--------------------+          +---------------------+          +---------------------+ 
```

We provide generic low-level filters/actions to users, and users can create customized filters/actions to wrap low-level
ones using cue.

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
- [ ] More Destination Types, such as WorkflowRun, API webhook, notifications(email, dingtalk, slack), execute velaql(
  CUE and K8s operations)
- [ ] Remove cache informer, make it with no catch but list watch events with unique queue.
- [ ] Make the configuration as CRD, launch new process/pod for new watcher
