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

// CommitSpec defines the desired state of Commit
type CommitSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Timeout defines the timeout for the NETCONF transaction
	// defaults to 1 seconds
	// +kubebuilder:default:=1
	Timeout int32 `json:"timeout,omitempty"`
	// If this Commit operation should occur after another operation, specify the other operation here.
	DependsOn DependsOn `json:"dependsOn,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Commit is the Schema for the commits API
type Commit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec      CommitSpec `json:"spec,omitempty"`
	RPCStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CommitList contains a list of Commit
type CommitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Commit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Commit{}, &CommitList{})
}

func (obj *Commit) GetMountPointNamespacedName(mountpoint string) string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: mountpoint}.String()
}

func (obj *Commit) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}
