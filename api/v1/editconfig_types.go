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

// EditConfigSpec defines the desired state of EditConfig
type EditConfigSpec struct {
	// Defines the NETCONF session to use
	MountPoint string `json:"mountPoint"`
	// Timeout defines the timeout for the NETCONF transaction
	// defaults to 1 seconds
	// +kubebuilder:default:=1
	Timeout int32 `json:"timeout,omitempty"`
	// Defined the operation to perform. Default to `merge`.
	//See https://datatracker.ietf.org/doc/html/rfc6241#section-7.2 for supported operations.
	// +kubebuilder:default:="merge"
	Operation string `json:"operation,omitempty"`
	// Identify the datastore against which the operation should be performed. Default to `candidate`.
	// +kubebuilder:default:="candidate"
	Target string `json:"target,omitempty"`
	// Define the XML payload to sent
	XML string `json:"xml"`
	// If this EditConfig operation should occur after another operation, specify the other operation here.
	DependsOn DependsOn `json:"dependsOn,omitempty"`
	// Whether to lock the specified datastore before doing the edit-config
	// +kubebuilder:default:=false
	Lock bool `json:"lock,omitempty"`
	// Whether to commit the changes.
	// +kubebuilder:default:=false
	Commit bool `json:"commit,omitempty"`
	// Whether to unlock the specified datastore before doing the edit-config
	// +kubebuilder:default:=false
	Unlock bool `json:"unlock,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EditConfig is the Schema for the editconfigs API
type EditConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec      EditConfigSpec `json:"spec,omitempty"`
	RPCStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EditConfigList contains a list of EditConfig
type EditConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EditConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EditConfig{}, &EditConfigList{})
}

func (obj *EditConfig) GetMountPointNamespacedName(mountpoint string) string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: mountpoint}.String()
}

func (obj *EditConfig) GetNamespacedName() string {
	return types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}.String()
}
