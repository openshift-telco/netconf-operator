package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DependsOn allows to specify a dependency for the operation to execute.
//If such dependency is not met, and/or if the underlying dependency isn't
//reporting success, the operation will fail.
type DependsOn struct {
	// Any of the Kind supported by netconf.openshift-telco.io/v1 Group
	Kind string `json:"kind,omitempty"`
	// The name of the object, which will be checked for within the same namespace
	Name string `json:"name,omitempty"`
}

func (d *DependsOn) IsNil() bool {
	return len(d.Kind) == 0 || len(d.Name) == 0
}

type KafkaSink struct {
	Enabled       bool   `json:"enabled"`
	Topic         string `json:"topic"`
	TransportType string `json:"transportType"`
	Broker        string `json:"broker"`
	Partition     int    `json:"partition"`
}

type RPCStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Either `success` or `failed`
	Status string `json:"status,omitempty"`
	// Provides the received RPC reply
	RpcReply string `json:"rpcReply,omitempty"`
	// Provide the list of supported capabilities
	Capabilities []string `json:"capabilities,omitempty"`
	// In case of a notification, keep track of the subscription-id
	SubscriptionID string `json:"subscriptionID,omitempty"`
}

func (obj *RPCStatus) GetConditions() []metav1.Condition {
	return obj.Conditions
}

func (obj *RPCStatus) SetConditions(reconcileStatus []metav1.Condition) {
	obj.Conditions = reconcileStatus
}
