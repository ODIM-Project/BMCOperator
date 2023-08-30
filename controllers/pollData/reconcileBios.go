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
	"path"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	controllers "github.com/ODIM-Project/BMCOperator/controllers/bmc"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
)

// AccommodateBiosDetails functions accommodates bios setting in object as per the changes in ODIM
func (r PollingReconciler) AccommodateBiosDetails(ctx context.Context, restClient restclient.RestClientInterface, mapOfSystems map[string]bool) {
	biosObj := &infraiov1.BiosSetting{}
	biosUtil := bios.GetBiosUtils(ctx, biosObj, r.commonRec, restClient, r.namespace)
	for system := range mapOfSystems {
		systemID := path.Base(system)
		r.AccommodateBiosInfo(ctx, biosUtil, systemID)
	}
}

func (r PollingReconciler) AccommodateBiosInfo(ctx context.Context, biosUtil bios.BiosInterface, systemID string) {
	r.bmcObject = r.commonRec.GetBmcObject(ctx, constants.StatusBmcSystemID, systemID, r.namespace)
	if r.bmcObject == nil {
		l.LogWithFields(ctx).Info("bmc object not found: ", systemID)
		return
	}
	if r.bmcObject.Status.BmcSystemID != "" {
		mapOfattributes := biosUtil.GetBiosAttributes(r.bmcObject)
		biosObject := r.commonRec.GetBiosObject(ctx, constants.MetadataName, r.bmcObject.ObjectMeta.Name, r.odimObj.Namespace)
		isequal, _ := utils.CompareMaps(mapOfattributes, biosObject.Status.BiosAttributes)
		if !isequal {
			r.commonRec.UpdateBiosSettingObject(ctx, mapOfattributes, biosObject)
		}
	}
}

// RevertBiosDetails function reverts back bios setting as per the object present in operator
func (r PollingReconciler) RevertBiosDetails(ctx context.Context, restClient restclient.RestClientInterface) {
	bmcObjs := r.commonRec.GetAllBmcObject(ctx, r.namespace)
	if bmcObjs != nil {
		for _, bmc := range *bmcObjs {
			r.CheckAndRevertBios(ctx, bmc, restClient)
		}
	}
}

func (r PollingReconciler) CheckAndRevertBios(ctx context.Context, bmc infraiov1.Bmc, restClient restclient.RestClientInterface) {
	if !common.RestartRequired[bmc.Status.BmcSystemID] {
		r.bmcObject = &bmc
		biosObj := r.commonRec.GetBiosObject(ctx, constants.MetadataName, r.bmcObject.Name, r.odimObj.Namespace)
		biosUtil := bios.GetBiosUtils(ctx, biosObj, r.commonRec, restClient, r.namespace)
		if bmc.Status.BiosAttributeRegistry != "" {
			mapOfattributes := biosUtil.GetBiosAttributes(r.bmcObject)
			if mapOfattributes != nil {
				isequal, mapOfBiosAttribute := utils.CompareMaps(biosObj.Status.BiosAttributes, mapOfattributes)
				if !isequal {
					biosObj.Spec.Bios = mapOfBiosAttribute
					ok, body := biosUtil.ValidateBiosAttributes(bmc.Status.BiosAttributeRegistry)
					if ok {
						if isUpdated := common.BiosAttributeUpdation(ctx, body, bmc.Status.BmcSystemID, restClient); isUpdated {
							common.RestartRequired[bmc.Status.BmcSystemID] = true
							go func() {
								bmc.Spec.BmcDetails.ResetType = "ForceRestart"
								bmcUtil := controllers.GetBmcUtils(ctx, &bmc, r.namespace, &r.commonRec, &restClient, common.GetCommonUtils(restClient), false)
								resetdone := bmcUtil.ResetSystem(true, true)
								if resetdone {
									attributes := biosUtil.GetBiosAttributes(r.bmcObject)
									biosObj.Spec.Bios = map[string]string{}
									biosObj.Status = infraiov1.BiosSettingStatus{
										BiosAttributes: attributes,
									}
									r.Client.Status().Update(ctx, biosObj)
								}
							}()
						}
					}
				}
			}
		}
	}
}
