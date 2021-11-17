/*
Copyright 2021. Alexis de Talhouët

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NotificationSpec defines the desired state of Notification
type NotificationSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Array of subscription to create
	CreateSubscription []CreateSubscription `json:"createSubscription,omitempty"`
	// Array of subscription to establish
	EstablishSubscription []EstablishSubscription `json:"establishSubscription,omitempty"`
}

// NotificationStatus defines the observed state of Notification
type NotificationStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Either `success` or `failed`
	Status string `json:"status,omitempty"`
	// Provides the received RPC reply
	RpcReply string `json:"rpcReply,omitempty"`
}

// CreateSubscription defines the desired state of CreateSubscription
type CreateSubscription struct {
	Name string `json:"name"`
	// Defines the stream to subscribe to. Defaults to NETCONF stream.
	// +kubebuilder:default:="NETCONF"
	Stream string `json:"stream,omitempty"`
	// Defines the start-time to listen to changes.
	StartTime string `json:"startTime,omitempty"`
	// Defines the time when to stop listen to changes.
	StopTime string `json:"stopTime,omitempty"`
}

// EstablishSubscription defines the desired state of EstablishSubscription
type EstablishSubscription struct {
	Name string `json:"name"`
	// Defines the `<establish-subscription` RPC to sent
	XML string `json:"xml"`
}

// GetIdentifier returns an unique identifier for the notification being registered.
func (obj *Notification) GetIdentifier(name string) string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: fmt.Sprintf("%s/%s", obj.Name, name)}.String()
}

func (obj *Notification) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}

func (obj *Notification) GetConditions() []metav1.Condition {
	return obj.Status.Conditions
}

func (obj *Notification) SetConditions(reconcileStatus []metav1.Condition) {
	obj.Status.Conditions = reconcileStatus
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Notification is the Schema for the notifications API
type Notification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationSpec   `json:"spec,omitempty"`
	Status NotificationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NotificationList contains a list of Notification
type NotificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notification `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notification{}, &NotificationList{})
}
