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

// DriveDetails defines the drive details for specific Array Controller
type DriveDetails struct {
	CapacityBytes string `json:"capacityBytes"`
	UsedInVolumes []int  `json:"usedInVolumes"`
}

// ArrayControllers defines the storage controllers for BMC
type ArrayControllers struct {
	SupportedRAIDLevel []string                `json:"supportedRAIDLevel"`
	Drives             map[string]DriveDetails `json:"drives,omitempty"`
}

// BMC defines a struct to hold basic properties of a BMC
type BMC struct {
	Address         string `json:"address"`
	ConnMethVariant string `json:"connectionMethodVariant"`
	PowerState      string `json:"powerState"`
	ResetType       string `json:"resetType"`
}

// Credential struct holds username and password for the BMC
type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// BmcSpec defines the desired state of Bmc
type BmcSpec struct {
	BmcDetails  BMC        `json:"bmc"`
	Credentials Credential `json:"credentials,omitempty"`
}

// BmcStatus defines the observed state of Bmc
type BmcStatus struct {
	BmcAddStatus          string                      `json:"bmcAddStatus"`
	BmcSystemID           string                      `json:"bmcSystemId"`
	SerialNumber          string                      `json:"serialNumber"`
	VendorName            string                      `json:"vendorName"`
	ModelID               string                      `json:"modelID"`
	FirmwareVersion       string                      `json:"firmwareVersion"`
	BiosVersion           string                      `json:"biosVersion"`
	BiosAttributeRegistry string                      `json:"biosAttributeRegistry"`
	EventsMessageRegistry string                      `json:"eventsMessageRegistry,omitempty"`
	StorageControllers    map[string]ArrayControllers `json:"storageControllers,omitempty"`
	SystemReset           string                      `json:"systemReset"`
	PowerState            string                      `json:"powerState"`
}

// SystemDetail struct defines basic properties of a system
type SystemDetail struct {
	Vendor  string
	ModelID string
}

// Vendor defines supportable vendors
var Vendor = []string{"HPE", "DELL", "LENOVO"}

// AllowableResetValues defines allowable reset values for a bmc
var AllowableResetValues = map[SystemDetail][]string{
	{"HPE", "ProLiant DL380 Gen10 Plus"}: {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"HPE", "ProLiant DL380 Gen10"}:      {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"HPE", "ProLiant DL360 Gen10"}:      {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"HPE", "ProLiant DL360 Gen10 Plus"}: {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"HPE", "ProLiant DL385 Gen10"}:      {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"HPE", "ProLiant e910"}:             {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"HPE", "ProLiant DL110 Gen10 Plus"}: {"On", "ForceOff", "GracefulShutdown", "ForceRestart", "Nmi", "PushPowerButton", "GracefulRestart"},
	{"DELL", "PowerEdge R440"}:           {"On", "ForceOff", "ForceRestart", "GracefulShutdown", "PushPowerButton", "Nmi", "PowerCycle"},
	{"LENOVO", "ThinkSystem SR630"}:      {"On", "Nmi", "GracefulShutdown", "GracefulRestart", "ForceOn", "ForceOff", "ForceRestart"},
}

// State defines the state to be applied by user
type State struct {
	PowerState        string
	DesiredPowerState string
	ResetType         string
}

// ResetWhenOnOn map defines if reset can be applied on bmc based on different state
var ResetWhenOnOn = map[State]bool{
	{"ON", "ON", "ForceRestart"}:     true,
	{"ON", "ON", "GracefulRestart"}:  true,
	{"ON", "ON", "GracefulShutdown"}: false,
	{"ON", "ON", "ForceOff"}:         false,
	{"ON", "ON", "On"}:               false,
	{"ON", "ON", "Pause"}:            true,
	{"ON", "ON", "Resume"}:           false,
	{"ON", "ON", "ForceOn"}:          false,
	{"ON", "ON", "Nmi"}:              true,
	{"ON", "ON", "PowerCycle"}:       true,
	{"ON", "ON", "PushPowerButton"}:  false,
	{"ON", "ON", "Suspend"}:          true,
}

// ResetWhenOffOff map defines if reset can be applied on bmc based on different state
var ResetWhenOffOff = map[State]bool{
	{"OFF", "OFF", "ForceRestart"}:     false,
	{"OFF", "OFF", "GracefulRestart"}:  false,
	{"OFF", "OFF", "GracefulShutdown"}: false,
	{"OFF", "OFF", "ForceOff"}:         false,
	{"OFF", "OFF", "On"}:               false,
	{"OFF", "OFF", "Pause"}:            false,
	{"OFF", "OFF", "Resume"}:           false,
	{"OFF", "OFF", "ForceOn"}:          false,
	{"OFF", "OFF", "Nmi"}:              false,
	{"OFF", "OFF", "PowerCycle"}:       false,
	{"OFF", "OFF", "PushPowerButton"}:  false,
	{"OFF", "OFF", "Suspend"}:          false,
}

// ResetWhenOnOff map defines if reset can be applied on bmc based on different state
var ResetWhenOnOff = map[State]bool{
	{"ON", "OFF", "ForceRestart"}:     false,
	{"ON", "OFF", "GracefulRestart"}:  false,
	{"ON", "OFF", "GracefulShutdown"}: true,
	{"ON", "OFF", "ForceOff"}:         true,
	{"ON", "OFF", "On"}:               false,
	{"ON", "OFF", "Pause"}:            true,
	{"ON", "OFF", "Resume"}:           false,
	{"ON", "OFF", "ForceOn"}:          false,
	{"ON", "OFF", "Nmi"}:              false,
	{"ON", "OFF", "PowerCycle"}:       false,
	{"ON", "OFF", "PushPowerButton"}:  true,
	{"ON", "OFF", "Suspend"}:          true,
}

// ResetWhenOffOn map defines if reset can be applied on bmc based on different state
var ResetWhenOffOn = map[State]bool{
	{"OFF", "ON", "ForceRestart"}:     false,
	{"OFF", "ON", "GracefulRestart"}:  false,
	{"OFF", "ON", "GracefulShutdown"}: false,
	{"OFF", "ON", "ForceOff"}:         false,
	{"OFF", "ON", "On"}:               true,
	{"OFF", "ON", "Pause"}:            false,
	{"OFF", "ON", "Resume"}:           false,
	{"OFF", "ON", "ForceOn"}:          true,
	{"OFF", "ON", "Nmi"}:              false,
	{"OFF", "ON", "PowerCycle"}:       true,
	{"OFF", "ON", "PushPowerButton"}:  true,
	{"OFF", "ON", "Suspend"}:          false,
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Bmc is the Schema for the bmcs API
// +kubebuilder:printcolumn:name="SystemID",type="string",JSONPath=".status.bmcSystemId"
// +kubebuilder:printcolumn:name="SerialNumber",type="string",JSONPath=".status.serialNumber"
// +kubebuilder:printcolumn:name="Vendor",type="string",JSONPath=".status.vendorName"
// +kubebuilder:printcolumn:name="ModelID",type="string",JSONPath=".status.modelID"
// +kubebuilder:printcolumn:name="FirmwareVersion",type="string",JSONPath=".status.firmwareVersion"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Bmc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BmcSpec   `json:"spec,omitempty"`
	Status BmcStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BmcList contains a list of Bmc
type BmcList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bmc `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bmc{}, &BmcList{})
}
