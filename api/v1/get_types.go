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

// GetSpec defines the desired state of Get
type GetSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Define the filter to apply; see more https://datatracker.ietf.org/doc/html/rfc6241#page-20
	// +kubebuilder:default:="subtree"
	FilterType string `json:"filterType,omitempty"`
	// Define the XML payload to sent
	XML string `json:"xml,omitempty"`
}

// GetStatus defines the observed state of Get
type GetStatus struct {
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

func (obj *Get) GetConditions() []metav1.Condition {
	return obj.Status.Conditions
}

func (obj *Get) SetConditions(reconcileStatus []metav1.Condition) {
	obj.Status.Conditions = reconcileStatus
}

func (obj *Get) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Get is the Schema for the gets API
type Get struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GetSpec   `json:"spec,omitempty"`
	Status GetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GetList contains a list of Get
type GetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Get `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Get{}, &GetList{})
}
