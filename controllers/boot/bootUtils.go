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

// BootOrderSetting defines boot order details
type BootOrderSetting struct {
	Boot Boot `json:"Boot,omitempty"`
}

// Boot defines boot order setting details
type Boot struct {
	BootOrder                    []string `json:"BootOrder,omitempty"`
	BootSourceOverrideEnabled    string   `json:"BootSourceOverrideEnabled,omitempty"`
	BootSourceOverrideTarget     string   `json:"BootSourceOverrideTarget,omitempty"`
	UefiTargetBootSourceOverride string   `json:"UefiTargetBootSourceOverride,omitempty"`
}

type BootInterface interface {
	GetBootAttributes(sysDetails map[string]interface{}) *infraiov1.BootSetting
	UpdateBootAttributesOnReset(bmcName string, bootSetting *infraiov1.BootSetting)
	updateBootSettings() bool
	UpdateBootDetails(ctx context.Context, systemID, bootBmcIP string, bootObj *infraiov1.BootOrderSetting, bootBmcObj *infraiov1.Bmc) bool
}

type bootUtils struct {
	ctx            context.Context
	bootObj        *infraiov1.BootOrderSetting
	commonRec      utils.ReconcilerInterface
	bootRestClient restclient.RestClientInterface
	commonUtil     common.CommonInterface
	namespace      string
}
