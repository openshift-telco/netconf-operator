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

// MountPointSpec defines the desired state of MountPoint
type MountPointSpec struct {

	// Represents the Netconf server to establish a session with.
	// By default, port `830` is used. If you need to use another port,
	// provide the target in the following format: `<host>:<port>`.
	Target string `json:"target"`
	// The username to use to authenticate
	Username string `json:"username"`
	// The password to use to authenticate
	Password string `json:"password"`
	// Whether to apply timeout while attempting the connection, expressed in second.
	// By default, set to `15` seconds. Put `0` for no timeout.
	// +kubebuilder:default:=15
	Timeout int64 `json:"timeout,omitempty"`
	// This is to instruct the NETCONF client to advertise additional capabilities
	AdditionalCapabilities []string `json:"additionalCapabilities,omitempty"`
}

// MountPointStatus defines the observed state of MountPoint
type MountPointStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Either `connected` or `failed`
	Status string `json:"status,omitempty"`
	// Provide the list of supported capabilities
	Capabilities []string `json:"capabilities,omitempty"`
}

func (obj *MountPoint) GetConditions() []metav1.Condition {
	return obj.Status.Conditions
}

func (obj *MountPoint) SetConditions(reconcileStatus []metav1.Condition) {
	obj.Status.Conditions = reconcileStatus
}

func (obj *MountPoint) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MountPoint is the Schema for the mountPoints API
type MountPoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MountPointSpec   `json:"spec,omitempty"`
	Status MountPointStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MountPointList contains a list of MountPoint
type MountPointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MountPoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MountPoint{}, &MountPointList{})
}
