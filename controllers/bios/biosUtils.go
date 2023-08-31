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
)

type BiosInterface interface {
	ValidateBiosAttributes(biosID string) (bool, map[string]interface{})
	getBiosLinkAndBody(resp map[string]interface{}, biosProps map[string]interface{}, biosBmcIP string) (string, []byte)
	UpdateBiosAttributesOnReset(biosBmcIP string, updatedBiosAttributes map[string]string)
	GetBiosAttributes(bmcObj *infraiov1.Bmc) map[string]string
	GetBiosAttributeID(biosVersion string, bmcObj *infraiov1.Bmc) (string, map[string]interface{})
}

type biosUtils struct {
	ctx            context.Context
	biosObj        *infraiov1.BiosSetting
	commonRec      utils.ReconcilerInterface
	biosRestClient restclient.RestClientInterface
	commonUtil     common.CommonInterface
	namespace      string
}
