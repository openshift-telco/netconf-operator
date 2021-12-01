# NETCONF operator

[![Report Card](https://goreportcard.com/badge/github.com/openshift-telco/netconf-operator)](https://goreportcard.com/report/github.com/openshift-telco/netconf-operator)
[![Build Status](https://travis-ci.org/adetalhouet/netconf-operator.png)](https://travis-ci.org/adetalhouet/netconf-operator)


This operator provides support for:
- [RFC6241](http://tools.ietf.org/html/rfc6241): **Network Configuration Protocol (NETCONF)**
    - Support for the following RPC: `lock`, `unlock`, `edit-config`, `comit`, `get`, `get-config`
    - Support for custom RPC
- [RFC6242](http://tools.ietf.org/html/rfc6242): **Using the NETCONF Protocol over Secure Shell (SSH)**
    - Support for username/password
    - No support for pub key 
- [RFC5277](https://datatracker.ietf.org/doc/html/rfc5277): **NETCONF Event Notifications**
    - Support for `create-subscription`
    - No support for notification filtering
- Partially [RFC8641](https://datatracker.ietf.org/doc/html/rfc8641) and [RFC8639](https://datatracker.ietf.org/doc/html/rfc8639): **Subscription to YANG Notifications for Datastore Updates**
    - Support for `establish-subscription`
    - Support for `delete-subscription`

The operator is built using the following [go-netconf](https://github.com/openshift-telco/go-netconf-client) client.

## RPC Usage

The `MountPoint` CRD is meant to establish an SSH connection to a remote NETCONF server.

All the below supported NETCONF operations depends on a `MountPoint` session to be established:
- `Get`
- `GetConfig`
- `EditConfig`
- `Commit`
- `Lock`
- `Unlock`
- `CreateSubscription`
- `EstablishSubscription`

All the CRDs, beside `EstablishSubscrption`, has no effect when deleted.

See the [examples](https://github.com/openshift-telco/netconf-operator/tree/main/examples) folder to understand how to use the CRD. Also, read the CRD spec to understand the requirements.

#### Sequence operations
In order to sequence operations, the `EditConfig`, `Commit`, and `Unlock` CRDs provide to ability to define an operation it is depending on, using the `dependsOn` field. As such, one can achieve such flow: `Lock` --> `EditConfig` --> `Commit` --> `Unlock`.

#### NETCONF notifications

By registering to a notification stream, the operator received the `notification` and translate it to a Kubernetes event. This enables the consumption of the events by downstream systems for further processing.
![](https://raw.githubusercontent.com/adetalhouet/netconf-operator/main/docs/netconf-notification-example.png)


##### Create subscription
When using the `create-subscription` CRD, only one NETCONF notification stream can be registered per session. Deleting a `CreateSubscription` CR has no effect. In order to remove that subscription, the RFC5277 stipulates to close the NETCONF session.

##### Establish subscription
There are no restriction on the `EstablishSubscription` CRD. It is mostly a wrapper to help manage notification handling. One session can handle many instance of the CR as using subscription will be uniquely identifiable by its _subscription-id_. When deleting a CR, the operator will execute a `delete-subscription` with the _subscription-id_ defined for that subscription.

## Dev

To build:
~~~
make docker-build docker-push IMG=quay.io/adetalho/netconf-operator:dev
~~~

To deploy:
~~~
make deploy IMG=quay.io/adetalho/netconf-operator:dev
~~~

To remove:
~~~
make undeploy
~~~

#### How the operator was generated using the Operator SDK

1 - create the scaffolding
~~~
operator-sdk init --domain=adetalhouet.io --repo=github.com/openshift-telco/netconf-operator
~~~
2. generate the netconf operations API.
~~~
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind Mountpoint
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind Commit
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind EditConfig
operator-sdk create api --resource=true --controller=true -group netconf --version v1 --kind GetConfig
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind Get
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind Lock
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind Unlock
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind RPC
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind RPC
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind CreateSubscription
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind EstablishSubscription
~~~


### Links

[Getting started with Operator SDK](https://docs.openshift.com/container-platform/4.8/operators/operator_sdk/golang/osdk-golang-quickstart.html)
