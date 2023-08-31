//(C) Copyright [2023] Hewlett Packard Enterprise Development LP
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

// AuthDetails struct contains information required for authentication
type AuthDetails struct {
	Password string `json:"password,omitempty"`
	Username string `json:"username,omitempty"`
}

// ImageDetails holds information for firmware image
type ImageDetails struct {
	ImageLocation string      `json:"imageLocation"`
	Auth          AuthDetails `json:"auth,omitempty"`
}

// FirmwareSpec defines the desired state of Firmware
type FirmwareSpec struct {
	Image            ImageDetails `json:"image"`
	TransferProtocol string       `json:"transferProtocol,omitempty"`
}

// FirmwareStatus defines the observed state of Firmware
type FirmwareStatus struct {
	Status          string `json:"status,omitempty"`
	FirmwareVersion string `json:"firmwareVersion,omitempty"`
	ImagePath       string `json:"imagePath,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Firmware is the Schema for the firmwares API
type Firmware struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FirmwareSpec   `json:"spec,omitempty"`
	Status FirmwareStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FirmwareList contains a list of Firmware
type FirmwareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Firmware `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Firmware{}, &FirmwareList{})
}
