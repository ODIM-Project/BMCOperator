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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EventsMessageRegistrySpec defines the desired state of EventsMessageRegistry
type EventsMessageRegistrySpec struct {
	Name            string                  `json:"Name,omitempty"`
	ID              string                  `json:"ID,omitempty"`
	OwningEntity    string                  `json:"OwningEntity,omitempty"`
	RegistryVersion string                  `json:"RegistryVersion,omitempty"`
	RegistryPrefix  string                  `json:"RegistryPrefix,omitempty"`
	Messages        map[string]EventMessage `json:"Messages,omitempty"`
}

// EventsMessageRegistryStatus defines the observed state of EventsMessageRegistry
type EventsMessageRegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// EventMessage defines the message struct
type EventMessage struct {
	Description  string         `json:"Description,omitempty"`
	Message      string         `json:"Message,omitempty"`
	NumberOfArgs string         `json:"NumberOfArgs,omitempty"`
	Oem          map[string]Oem `json:"Oem,omitempty"`
	ParamTypes   []string       `json:"ParamTypes,omitempty"`
	Resolution   string         `json:"Resolution,omitempty"`
	Severity     string         `json:"Severity,omitempty"`
}

// Oem defines the Oem property struct
type Oem struct {
	OdataType      string `json:"odataType,omitempty"`
	HealthCategory string `json:"HealthCategory,omitempty"`
	Type           string `json:"Type,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EventsMessageRegistry is the Schema for the eventsmessageregistries API
type EventsMessageRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventsMessageRegistrySpec   `json:"spec,omitempty"`
	Status EventsMessageRegistryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventsMessageRegistryList contains a list of EventsMessageRegistry
type EventsMessageRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventsMessageRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventsMessageRegistry{}, &EventsMessageRegistryList{})
}
