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
	bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
)

type firmwareUtils struct {
	ctx            context.Context
	firmObj        *infraiov1.Firmware
	commonRec      utils.ReconcilerInterface
	firmRestClient restclient.RestClientInterface
	commonUtil     common.CommonInterface
	biosUtil       bios.BiosInterface
	namespace      string
}

type firmwareInterface interface {
	getFirmwareBody() ([]byte, error)
	UpdateFirmwareStatus(string, string, string)
	GetSystemID() string
	GetFirmwareVersion() string
	UpdateFirmwareLabels(string)
	UpdateBmcObjectWithFirmwareVersionAndSchema(*infraiov1.Bmc, string, string)
	CreateFirmwareInSystem([]byte) (bool, bool, error)
	checkIfBiosSchemaRegistryChanged(bmcObj *infraiov1.Bmc) string
	GetFirmwareInventoryDetails(systemID string) []string
}

type firmPayload struct {
	Image            string   `json:"ImageURI"`
	Targets          []string `json:"Targets"`
	Username         string   `json:"Username"`
	Password         string   `json:"Password"`
	TransferProtocol string   `json:"transferProtocol"`
}

type firmPayloadWithTransferProtocol struct {
	Image            string   `json:"ImageURI"`
	Targets          []string `json:"Targets"`
	Username         string   `json:"Username"`
	Password         string   `json:"Password"`
	TransferProtocol string   `json:"transferProtocol"`
}

type FirmPayloadWithoutAuth struct {
	Image   string   `json:"ImageURI"`
	Targets []string `json:"Targets"`
}
