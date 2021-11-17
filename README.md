# NETCONF operator

[![Report Card](https://goreportcard.com/badge/github.com/adetalhouet/netconf-operator)](https://goreportcard.com/report/github.com/adetalhouet/netconf-operator)
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
    - No support for `delete-subscription`
It is build using the following [go-netconf](https://github.com/adetalhouet/go-netconf) implementation.

## RPC Usage

The `MountPoint` CRD is meant to establish an SSH connection to a remote NETCONF server.

All the below supported NETCONF operations depends on a `MountPoint` session to be established:
- `Get`
- `GetConfig`
- `EditConfig`
- `Commit`
- `Lock`
- `Unlock`

See the [examples]() folder to understand how to use the CRD. Also, read the CRD spec to understand the requirements.

The `Lock` CRD removes the lock on the datastore when deleted; so removal of a `Lock` CR acts like as an unlock.

Finally, in order to sequence operations, the `EditConfig`, `Commit`, and `Unlock` CRDs provide to ability to define an operation it is depending on, using the `dependsOn` field. As such, one can achieve such flow: `Lock` --> `EditConfig` --> `Commit` --> `Unlock`.

## Notification Usage

The `Notification` CRD enables the creation of `create-subscription` and `establish-subscription` RPC. One `Notification` CR can provide multiple instead of each, enabling an easy way to setup telemetry. All the received notifications are translated to a Kubernetes Event, providing the received XML for further analysis.

## TODO
- enable `commit` through `edit-config` directly
- implement proper CRD cleanup sequence and dependency
- add support for `delete-subscription`

## Dev

To build:
~~~
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
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
operator-sdk init --domain=adetalhouet.io --repo=github.com/adetalhouet/netconf-operator
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
operator-sdk create api --resource=true --controller=true --group netconf --version v1 --kind Notification
~~~


### Links

[Getting started with Operator SDK](https://docs.openshift.com/container-platform/4.8/operators/operator_sdk/golang/osdk-golang-quickstart.html)
