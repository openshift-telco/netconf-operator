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

// RPCSpec defines the desired state of RPC
type RPCSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Timeout defines the timeout for the NETCONF transaction
	// defaults to 1 seconds
	// +kubebuilder:default:=1
	Timeout int32 `json:"timeout,omitempty"`
	// Define the XML payload to sent
	XML string `json:"xml"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RPC is the Schema for the rpcs API
type RPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec      RPCSpec `json:"spec,omitempty"`
	RPCStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RPCList contains a list of RPC
type RPCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RPC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RPC{}, &RPCList{})
}

func (obj *RPC) GetMountPointNamespacedName(mountpoint string) string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: mountpoint}.String()
}

func (obj *RPC) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}
