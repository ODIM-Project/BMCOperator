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

// BootOrderSettingsSpec defines the desired state of BootOrderSettings
type BootOrderSettingsSpec struct {
	BmcName  string       `json:"bmcName,omitempty"`
	SystemID string       `json:"systemID,omitempty"`
	SerialNo string       `json:"serialNumber,omitempty"`
	Boot     *BootSetting `json:"boot,omitempty"`
}

// BootOrderSettingsStatus defines the observed state of BootOrderSettings
type BootOrderSettingsStatus struct {
	Boot BootSetting `json:"boot,omitempty"`
}

// BootSetting defines the different settings for boot
type BootSetting struct {
	BootOrder                    []string `json:"bootOrder,omitempty"`
	BootSourceOverrideEnabled    string   `json:"bootSourceOverrideEnabled,omitempty"`
	BootSourceOverrideMode       string   `json:"bootSourceOverrideMode,omitempty"`
	BootSourceOverrideTarget     string   `json:"bootSourceOverrideTarget,omitempty"`
	BootTargetAllowableValues    []string `json:"bootSourceOverrideTarget.AllowableValues,omitempty"`
	UefiTargetBootSourceOverride string   `json:"uefiTargetBootSourceOverride,omitempty"`
	UefiTargetAllowableValues    []string `json:"uefiTargetBootSourceOverride.AllowableValues,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BootOrderSetting is the Schema for the bootordersetting API
type BootOrderSetting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BootOrderSettingsSpec   `json:"spec,omitempty"`
	Status BootOrderSettingsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BootOrderSettingList contains a list of BootOrderSettings
type BootOrderSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BootOrderSetting `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BootOrderSetting{}, &BootOrderSettingList{})
}
