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

// UnlockSpec defines the desired state of Unlock
type UnlockSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Identify the datastore against which the operation should be performed. Default to `candidate`.
	// +kubebuilder:default:="candidate"
	Target string `json:"target,omitempty"`
	// If this Unlock operation should occur after another operation, specify the other operation here.
	DependsOn DependsOn `json:"dependsOn,omitempty"`
}

// UnlockStatus defines the observed state of Unlock
type UnlockStatus struct {
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

func (obj *Unlock) GetConditions() []metav1.Condition {
	return obj.Status.Conditions
}

func (obj *Unlock) SetConditions(reconcileStatus []metav1.Condition) {
	obj.Status.Conditions = reconcileStatus
}

func (obj *Unlock) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Unlock is the Schema for the unlocks API
type Unlock struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UnlockSpec   `json:"spec,omitempty"`
	Status UnlockStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UnlockList contains a list of Unlock
type UnlockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Unlock `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Unlock{}, &UnlockList{})
}
