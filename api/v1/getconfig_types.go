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

// GetConfigSpec defines the desired state of GetConfig
type GetConfigSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Identify the datastore against which the operation should be performed. Default to `running`.
	// +kubebuilder:default:="running"
	Target string `json:"target,omitempty"`
}

// GetConfigStatus defines the observed state of GetConfig
type GetConfigStatus struct {
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

func (obj *GetConfig) GetConditions() []metav1.Condition {
	return obj.Status.Conditions
}

func (obj *GetConfig) SetConditions(reconcileStatus []metav1.Condition) {
	obj.Status.Conditions = reconcileStatus
}

func (obj *GetConfig) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GetConfig is the Schema for the getconfigs API
type GetConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GetConfigSpec   `json:"spec,omitempty"`
	Status GetConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GetConfigList contains a list of GetConfig
type GetConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GetConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GetConfig{}, &GetConfigList{})
}
