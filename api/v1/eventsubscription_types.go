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

// EventsubscriptionSpec defines the desired state of Eventsubscription
type EventsubscriptionSpec struct {
	Name                 string   `json:"name,omitempty"`
	Destination          string   `json:"destination"`
	EventTypes           []string `json:"eventTypes,omitempty"`
	MessageIds           []string `json:"messageIds,omitempty"`
	ResourceTypes        []string `json:"resourceTypes,omitempty"`
	Context              string   `json:"context"`
	EventFormatType      string   `json:"eventFormatType,omitempty"`
	SubordinateResources bool     `json:"subordinateResources,omitempty"`
	OriginResources      []string `json:"originResources,omitempty"`
}

// EventsubscriptionStatus defines the observed state of Eventsubscription
type EventsubscriptionStatus struct {
	ID               string   `json:"eventSubscriptionID"`
	Name             string   `json:"name,omitempty"`
	Destination      string   `json:"destination"`
	Context          string   `json:"context"`
	Protocol         string   `json:"protocol"`
	EventTypes       []string `json:"eventTypes,omitempty"`
	MessageIds       []string `json:"messageIds,omitempty"`
	SubscriptionType string   `json:"subscriptionType"`
	ResourceTypes    []string `json:"resourceTypes,omitempty"`
	OriginResources  []string `json:"originResources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Eventsubscription is the Schema for the eventsubscriptions API
// +kubebuilder:printcolumn:name="EventSubscriptionID",type="string",JSONPath=".status.eventSubscriptionID"
// +kubebuilder:printcolumn:name="Destination",type="string",JSONPath=".status.destination"
// +kubebuilder:printcolumn:name="Context",type="string",JSONPath=".status.context"
// +kubebuilder:printcolumn:name="SubscriptionType",type="string",JSONPath=".status.subscriptionType"
type Eventsubscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventsubscriptionSpec   `json:"spec,omitempty"`
	Status EventsubscriptionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventsubscriptionList contains a list of Eventsubscription
type EventsubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Eventsubscription `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Eventsubscription{}, &EventsubscriptionList{})
}
