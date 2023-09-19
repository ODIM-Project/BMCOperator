//(C) Copyright [2022] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VolumeSpec defines the desired state of Volume
// +kubebuilder:validation:XPreserveUnknownFields
type VolumeSpec struct {
	StorageControllerID string   `json:"storageControllerID,omitempty"`
	RAIDType            string   `json:"RAIDType,omitempty"`
	Drives              []string `json:"drives,omitempty"`
}

type Identifier struct {
	DurableName       string `json:"DurableName"`
	DurableNameFormat string `json:"DurableNameFormat"`
}

// VolumeStatus defines the observed state of Volume
// +kubebuilder:validation:XPreserveUnknownFields
type VolumeStatus struct {
	VolumeID            string     `json:"volumeID"`
	VolumeName          string     `json:"volumeName"`
	RAIDType            string     `json:"RAIDType"`
	StorageControllerID string     `json:"storageControllerID"`
	Drives              []string   `json:"drives"`
	CapacityBytes       string     `json:"capacityBytes"`
	Identifiers         Identifier `json:"Identifiers"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Volume is the Schema for the volumes API
// +kubebuilder:printcolumn:name="VolumeName",type="string",JSONPath=".status.volumeName"
// +kubebuilder:printcolumn:name="VolumeID",type="string",JSONPath=".status.volumeID"
// +kubebuilder:printcolumn:name="RAIDType",type="string",JSONPath=".status.RAIDType"
// +kubebuilder:printcolumn:name="CapacityBytes",type="string",JSONPath=".status.capacityBytes"
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VolumeSpec   `json:"spec,omitempty"`
	Status            VolumeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// VolumeList contains a list of Volume
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}
