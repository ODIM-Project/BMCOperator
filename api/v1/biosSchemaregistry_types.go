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

// BiosSchemaRegistrySpec defines the desired state of BiosSchemaRegistry
type BiosSchemaRegistrySpec struct {
	Name             string              `json:"Name,omitempty"`
	ID               string              `json:"ID,omitempty"`
	OwningEntity     string              `json:"OwningEntity,omitempty"`
	RegistryVersion  string              `json:"RegistryVersion,omitempty"`
	Attributes       []map[string]string `json:"Attributes,omitempty"`
	SupportedSystems []SupportedSystems  `json:"SupportedSystems,omitempty"`
}

// BiosSchemaRegistryStatus defines the observed state of BiosSchemaRegistry
type BiosSchemaRegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// SupportedSystems defines all the supported system for schema
type SupportedSystems struct {
	ProductName     string `json:"ProductName,omitempty"`
	SystemID        string `json:"SystemId,omitempty"`
	FirmwareVersion string `json:"FirmwareVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BiosSchemaRegistry is the Schema for the biosschemaregistries API
type BiosSchemaRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BiosSchemaRegistrySpec   `json:"spec,omitempty"`
	Status BiosSchemaRegistryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BiosSchemaRegistryList contains a list of BiosSchemaRegistry
type BiosSchemaRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BiosSchemaRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BiosSchemaRegistry{}, &BiosSchemaRegistryList{})
}
