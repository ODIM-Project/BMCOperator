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

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	"k8s.io/apimachinery/pkg/types"
)

// VolFinalizer: finalizer for volume object
const VolFinalizer = "infra.io.volume/finalizer"

// VolumeInterface declares method signatures to be defined by Volume utils
type VolumeInterface interface {
	CreateVolume(bmcObj *infraiov1.Bmc, dispName string, namespacedName types.NamespacedName, updateVolumeObject bool, commonRec utils.ReconcilerInterface) (bool, error)
	GetVolumeRequestPayload(systemID, dispName string) []byte
	UpdateVolumeStatusAndClearSpec(bmcObj *infraiov1.Bmc, dispName string) (bool, string)
	DeleteVolume(bmcObj *infraiov1.Bmc) bool
	UpdateVolumeObject(volObj *infraiov1.Volume)
}

type volumeUtils struct {
	ctx              context.Context
	volumeRestClient restclient.RestClientInterface
	commonRec        utils.ReconcilerInterface
	commonUtil       common.CommonInterface
	volObj           *infraiov1.Volume
	namespace        string
}

// volume request payload struct
type requestPayload struct {
	RAIDType    string `json:"RAIDType"`
	Links       link   `json:"Links"`
	DisplayName string `json:"DisplayName"`
	ApplyTime   string `json:"@Redfish.OperationApplyTime,omitempty"`
}

type link struct {
	Drives []map[string]string `json:"Drives"`
}
