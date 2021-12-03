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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateSubscriptionSpec defines the desired state of CreateSubscription
type CreateSubscriptionSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Timeout defines the timeout for the NETCONF transaction
	// defaults to 1 seconds
	// +kubebuilder:default:=1
	Timeout int32 `json:"timeout,omitempty"`
	// Defines the stream to subscribe to
	Stream string `json:"stream,omitempty"`
	// Defines the start-time to listen to changes.
	StartTime string `json:"startTime,omitempty"`
	// Defines the time when to stop listen to changes.
	StopTime string `json:"stopTime,omitempty"`
	// Used to forward received notification to kafka
	KafkaSink KafkaSink `json:"kafkaSink,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CreateSubscription is the Schema for the createsubscriptions API
type CreateSubscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec      CreateSubscriptionSpec `json:"spec,omitempty"`
	RPCStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CreateSubscriptionList contains a list of CreateSubscription
type CreateSubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CreateSubscription `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CreateSubscription{}, &CreateSubscriptionList{})
}

func (obj *CreateSubscription) GetMountPointNamespacedName(mountpoint string) string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: mountpoint}.String()
}

func (obj *CreateSubscription) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}
