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

package constants

const (
	// FirmwareSimpleUpdateURI : uri of firmware simple update API
	FirmwareSimpleUpdateURI = "/redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate"
	// FirmwareStartUpdateURI : uri of firmware start update API
	FirmwareStartUpdateURI = "/redfish/v1/UpdateService/Actions/UpdateService.StartUpdate"
	// BMCOPERATOR : bmc-operator folder name
	BMCOPERATOR = "BMCOperator"
	// DefaultTimeTicker defines the default time set for ticker
	DefaultTimeTicker = 24
	// SystemURI defines the URI for systems collection
	SystemURI = "/redfish/v1/Systems/"
	// Firmware contains string for firmware kind
	Firmware = "Firmware"
	// Accommodate contails string for Accommodate poll Action
	Accommodate = "Accommodate"
	// Revert contails string for Revert poll Action
	Revert = "Revert"
	// ConfigMapName
	ConfigMapName = "config"
	// ILOManufacturer is the manufacturer name for iLO BMC
	ILOManufacturer = "HPE"

	SleepTime           = 30 //sec
	RetryCount          = 10
	PluginSyncInSeconds = 120 //sec

	// Used for indexing in List()
	MetadataName              = "metadata.name"
	StatusSerialNumber        = "status.serialNumber"
	SpecBmcAddress            = "spec.bmc.address"
	StatusBmcSystemID         = "status.bmcSystemId"
	StatusEventsubscriptionID = "status.eventSubscriptionID"

	// systemReset event
	PendingForResetEvent = "Pending for"

	//Volume request details
	ApplyTime = "OnReset"

	// EventsubscriptionFinalizer is the finalizer for eventsubscription
	EventsubscriptionFinalizer = "infra.io.eventsubscription/finalizer"

	//Logging operation ActionID and ActionName
	BmcOperator                   = "bmc-operator"
	BMCSettingActionID            = "001"
	BMCSettingActionName          = "BMCOperations"
	BIOSSettingActionID           = "002"
	BIOSSettingActionName         = "BIOSSettings"
	BootOrderActionID             = "003"
	BootOrderActionName           = "BootOrderSettings"
	VolumeActionID                = "004"
	VolumeActionName              = "VolumeOperations"
	ODIMObjectOperationActionID   = "000"
	ODIMObjectOperationActionName = "ODIMRegistration"
	PollingActionID               = "006"
	PollingActionName             = "GetPollingData"
	EventClientActionID           = "006"
	EventClientActionName         = "EventClient"
	EventSubscriptionActionName   = "EventSubscription"
	EventSubscriptionActionID     = "007"
	TrackFileConfigActionName     = "TrackConfigChanges"
	TrackFileConfigActionID       = "008"
)
