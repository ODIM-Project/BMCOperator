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

package controllers

import (
	"context"
	"net/http"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	v1 "github.com/ODIM-Project/BMCOperator/api/v1"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
)

// defines the operations
const (
	ADDBMC                  = "AddBMC"
	DELETEBMC               = "DeleteBMC"
	UPDATEBMC               = "UpdateBMC"
	RESETBMC                = "ResetBMC"
	BOOTSETTING             = "BootSetting"
	BIOSSETTING             = "BiosSetting"
	CREATEVOLUME            = "CreateVolume"
	DELETEVOLUME            = "DeleteVolume"
	FIRMWARE                = "Firmware"
	EVENTSUBSCRIPTION       = "CreateEventSubscription"
	DELETEEVENTSUBSCRIPTION = "DeleteEventSubscription"
)

var (
	KubeConfigPath string
)

type CommonInterface interface {
	GetBmcSystemDetails(context.Context, *infraiov1.Bmc) map[string]interface{}
	MoniteringTaskmon(headerInfo http.Header, ctx context.Context, operation, resourceName string) (bool, map[string]interface{})
	BmcAddition(ctx context.Context, bmcObject *v1.Bmc, body []byte, restClient restclient.RestClientInterface) (bool, map[string]interface{})
	BmcDeleteOperation(ctx context.Context, aggregationURL string, restClient restclient.RestClientInterface, resourceName string) bool
}

type CommonUtils struct {
	restClient restclient.RestClientInterface
}

// ComputerSystemReset struct is used for creating reset request
type ComputerSystemReset struct {
	ResetType string
}

// MapOfBmc contains list of bmc's with there current state
var MapOfBmc = make(map[string]BmcState)

// trackVolumeStatus will tell us the current state of volume
// Defines volume of each storage controller of specific bmc
// {"bmc1,ArrayController-0,1":{true,false},"bmc1,ArrayController-0,2":{true,false},"ArrayController-1":VolumeStatus{"1":{true,false},"2":{false,true},...},..},...}
var TrackVolumeStatus = map[BmcStorageControllerVolumesStatus]VolumeStatus{}

// BmcStorageControllerVolumesStatus stores volume status for each storageController for each volume for each bmc
type BmcStorageControllerVolumesStatus struct {
	Bmc               string
	StorageController string
	VolumeID          string
}

// RestartRequired is to check if restart is required or not
var RestartRequired = make(map[string]bool)

// MapOfFirmware defines the state of firmware update
var MapOfFirmware = make(map[string]bool)

// BmcState defines current state of BMC
type BmcState struct {
	IsAdded      bool
	IsDeleted    bool
	IsObjCreated bool
}

// VolumeStatus stores the status for each volume
type VolumeStatus struct {
	IsGettingCreated bool
	IsGettingDeleted bool
}

var TaskmonLogsTable = map[string]string{
	"201:AddBMC":                  "BMC %s is successfully added, updating bmc object...",
	"202:AddBMC":                  "BMC %s addition is ongoing...",
	"409:AddBMC":                  "%s BMC already added!!",
	"Err:AddBMC":                  "error while adding %s BMC",
	"200:ResetBMC":                "BMC %s reset is successful, updating bmc object...",
	"202:ResetBMC":                "BMC %s reset ongoing...",
	"404:ResetBMC":                "%s BMC not present",
	"400:ResetBMC":                "Invalid request passed while resetting %s BMC",
	"409:ResetBMC":                "Cannot reset %s BMC, value conflicting",
	"Err:ResetBMC":                "error while performing reset for %s BMC",
	"204:DeleteBMC":               "BMC %s deletion is successful",
	"202:DeleteBMC":               "BMC %s deletion inprogress...",
	"404:DeleteBMC":               "%s BMC is not added to ODIM, forbidding delete on it",
	"409:DeleteBMC":               "Already a delete request is in progress for %s BMC",
	"Err:DeleteBMC":               "error while deleting %s BMC",
	"202:UpdateBMC":               "BMC %s password updation inprogress...",
	"200:UpdateBMC":               "Updated new password in %s BMC",
	"401:UpdateBMC":               "Password is not authorised for %s BMC",
	"Err:UpdateBMC":               "error while updating password for %s BMC",
	"200:BootSetting":             "Boot order setting configured for %s BMC, updating boot order setting object...",
	"202:BootSetting":             "Configuring boot order setting for %s BMC...",
	"404:BootSetting":             "Cannot configure boot order setting for %s BMC, since %s BMC is not found!",
	"400:BootSetting":             "Invalid request passed while configuring boot setting %s BMC!",
	"Err:BootSetting":             "Error while configuring boot order setting for %s BMC",
	"200:BiosSetting":             "Bios setting configured for %s BMC, updating bios order setting object...",
	"202:BiosSetting":             "Configuring bios setting for %s BMC...",
	"404:BiosSetting":             "Cannot configure bios setting for %s BMC, since %s BMC is not found!",
	"400:BiosSetting":             "Invalid request passed while configuring bios setting %s BMC!",
	"Err:BiosSetting":             "Error in patching, bios not configured properly for %s BMC, try again",
	"200:CreateVolume":            "Volume %s successfully created, Please reset the system now",
	"202:CreateVolume":            "%s volume creation in progress... ",
	"Err:CreateVolume":            "error while creating %s volume",
	"400:CreateVolume":            "Invalid request passed while creating volume for %s! ",
	"404:CreateVolume":            "The drives requested for %s volume creation is not present",
	"204:DeleteVolume":            "Volume %s successfully deleted, Please reset the system now",
	"202:DeleteVolume":            "%s volume deletion in progress... ",
	"404:DeleteVolume":            "%s volume does not exist, hence deletion is not possible",
	"Err:DeleteVolume":            "error while deleteing %s volume",
	"200:Firmware":                "Firmware update completed for %s BMC, updating firmware object...",
	"400:Firmware":                "Firmware update unsuccessful for %s BMC, try again",
	"404:Firmware":                "Firmware update unsuccessful for %s BMC, try again",
	"202:Firmware":                "Firmware update in progress for %s BMC...",
	"Err:Firmware":                "error while applying firmware for %s BMC",
	"202:CreateEventSubscription": "EventSubscription %s creation in progress",
	"201:CreateEventSubscription": "EventSubscription %s created successfully",
	"400:CreateEventSubscription": "Invalid request for creating %s eventSubscription",
	"409:CreateEventSubscription": "Subscription already exists with destination %s",
	"200:DeleteEventSubscription": "Eventsubscription %s deletion is successful",
}
