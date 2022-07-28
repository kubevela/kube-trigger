# kube-trigger

> This project is still in development.

kube-trigger can list and watch kubernetes object event and trigger an event to destination. The project is inspired by [kubewatch](https://github.com/vmware-archive/kubewatch).

Currently, the basic usage of kube-trigger is to watch any kind of Kubernetes CRD and trigger update of Application. It can solve issues like https://github.com/kubevela/kubevela/issues/4418 .

But the usage of kube-trigger is more than that, actually it's a lightweight event-trigger in Kubernetes world. The architecture can be:

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

## TODO:
- [ ] Dockerfile, Makefile, linter, GitHub Actions
- [ ] Make it run as Addon, build component definition, and examples
- [ ] Notification for more than one app: selector from compose of Namespace; Labels; Name
- [ ] Refine README
- [ ] More Source types, support read from VelaQL
- [ ] More Destination Types, such as WorkflowRun, API webhook, notifications(email, dingtalk, slack), execute velaql(CUE and K8s operations)
- [ ] Remove cache informer, make it with no catch but list watch events with unique queue.
- [ ] Make the configuration as CRD, launch new process/pod for new watcher
