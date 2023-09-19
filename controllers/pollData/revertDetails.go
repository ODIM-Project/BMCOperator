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

// Package controllers ...
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"

	v1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	controllers "github.com/ODIM-Project/BMCOperator/controllers/bmc"
	boot "github.com/ODIM-Project/BMCOperator/controllers/boot"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	firmware "github.com/ODIM-Project/BMCOperator/controllers/firmware"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
)

// RevertPollingDetails reverts the changes on bmc
func (r PollingReconciler) RevertPollingDetails(ctx context.Context, restClient restclient.RestClientInterface, mapOfSystems map[string]bool) {
	mapOfBmcObj := make(map[string]bool)
	isMarkedDeleted := make(map[string]bool)
	// get all bmc object
	bmcObjs := r.commonRec.GetAllBmcObject(ctx, config.Data.Namespace)
	// if bmcOjbects are not present then skip
	if bmcObjs != nil {
		// loop over bmc objects present in operator
		for _, bmc := range *bmcObjs {
			r.bmcObject = &bmc
			if r.checkIfSystemIsUndergoingReset() {
				continue
			}
			// if deletion process is in progress then mark the device as isMarkedDeleted
			if bmc.GetDeletionTimestamp() != nil {
				isMarkedDeleted[constants.SystemURI+bmc.Status.BmcSystemID] = true
			}
			// add all the bmc's present in operator to mapOfBmcObj
			mapOfBmcObj[constants.SystemURI+bmc.Status.BmcSystemID] = true
			if !isMarkedDeleted[constants.SystemURI+bmc.Status.BmcSystemID] {
				if ok := mapOfSystems[constants.SystemURI+bmc.Status.BmcSystemID]; !ok {
					r.bmcObject = &bmc
					isAdded, id := r.bmcAdditionOperation(ctx, restClient, bmc.Status.BmcSystemID)
					if isAdded && id != "" {
						r.bmcObject.Labels["systemId"] = id
						err := r.Client.Update(ctx, r.bmcObject)
						if err != nil {
							l.LogWithFields(ctx).Errorf("error while updating bmc object %s", err)
						}
						r.bmcObject.Status.BmcSystemID = id
						// update latest bmc id in bmc object
						updateErr := r.commonRec.GetCommonReconcilerClient().Status().Update(ctx, r.bmcObject)
						if updateErr != nil {
							l.LogWithFields(ctx).Errorf("error while updating object status %s", updateErr)
						}
						mapOfSystems[constants.SystemURI+id] = true
						// delete the key from map since new id gets generated once bmc is added to odim
						delete(mapOfBmcObj, constants.SystemURI+bmc.Status.BmcSystemID)
						mapOfBmcObj[constants.SystemURI+id] = true
					}
				}
			}
			// TODO: move the below functions to different file
			if ok := common.MapOfFirmware[bmc.GetName()]; !ok {
				r.CheckAndRevertFirmwareVersion(ctx, bmc, restClient)
				r.CheckAndRevertBoot(ctx, bmc, restClient)
			}

		}
	}
	//delete systems from odim if no object present
	for key := range mapOfSystems {
		systemID := path.Base(key)
		r.bmcObject = r.commonRec.GetBmcObject(ctx, constants.StatusBmcSystemID, systemID, r.namespace)
		if r.bmcObject != nil && r.checkIfSystemIsUndergoingReset() {
			continue
		}
		if !mapOfBmcObj[key] && !isMarkedDeleted[key] && !mapOfBmcObj[constants.SystemURI] {
			aggregationURL := strings.Replace(key, "/Systems/", "/AggregationService/AggregationSources/", 1)
			r.commonUtil.BmcDeleteOperation(ctx, aggregationURL, r.bmcObject.ObjectMeta.Name)
		}
		delete(mapOfBmcObj, constants.SystemURI)
	}
}

// RevertResourceRemoved is called when event resource removed is arrived
func (r *PollingReconciler) RevertResourceRemoved(ctx context.Context, systemURL string) {
	isMarkedDeleted := make(map[string]bool)
	// get all bmc object
	systemID := path.Base(systemURL)
	r.bmcObject = r.commonRec.GetBmcObject(ctx, constants.StatusBmcSystemID, systemID, r.namespace)
	// if bmcOjbects are not present then skip
	if r.bmcObject != nil {
		if r.checkIfSystemIsUndergoingReset() {
			return
		}
		// if deletion process is in progress then mark the device as isMarkedDeleted
		if r.bmcObject.GetDeletionTimestamp() != nil {
			isMarkedDeleted[constants.SystemURI+r.bmcObject.Status.BmcSystemID] = true
		}
		// add all the bmc's present in operator to mapOfBmcObj
		if !isMarkedDeleted[constants.SystemURI+r.bmcObject.Status.BmcSystemID] {
			if systemID == r.bmcObject.Status.BmcSystemID {
				isAdded, id := r.bmcAdditionOperation(ctx, r.pollRestClient, r.bmcObject.Status.BmcSystemID)
				if isAdded && id != "" {
					r.bmcObject.Labels["systemId"] = id
					err := r.Client.Update(ctx, r.bmcObject)
					if err != nil {
						l.LogWithFields(ctx).Errorf("error while updating bmc object %s", err)
					}
					r.bmcObject.Status.BmcSystemID = id
					// update latest bmc id in bmc object
					updateErr := r.commonRec.GetCommonReconcilerClient().Status().Update(ctx, r.bmcObject)
					if updateErr != nil {
						l.LogWithFields(ctx).Errorf("error while updating object status %s", updateErr)
					}
				}
			}

		}
		// TODO: move the below functions to different file
		if ok := common.MapOfFirmware[r.bmcObject.GetName()]; !ok {
			r.CheckAndRevertFirmwareVersion(ctx, *r.bmcObject, r.pollRestClient)
			r.CheckAndRevertBoot(ctx, *r.bmcObject, r.pollRestClient)
		}
	}
}

// RevertResourceAdded is called when event resource removed is arrived
func (r *PollingReconciler) RevertResourceAdded(ctx context.Context, systemURL string) {

	aggregationURL := strings.Replace(systemURL, "/Systems/", "/AggregationService/AggregationSources/", 1)
	systemID := path.Base(systemURL)
	var resp map[string]interface{}
	// checking if the aggregation source has been completely added so that it does find the object on operator
	for i := 0; i < 3; i++ {
		var statusCode int
		resp, statusCode, _ = r.pollRestClient.Get(aggregationURL, "Checking BMC addition")
		l.LogWithFields(r.ctx).Debugf("Response from GET on %s:%v", aggregationURL, resp)
		if statusCode == http.StatusOK {
			break
		}

		l.LogWithFields(ctx).Debugf("Aggregation source has not been added successfully waiting ...")

		time.Sleep(20 * time.Second)
	}
	if Hostname, ok := resp["HostName"]; ok {
		r.bmcObject = r.commonRec.GetBmcObject(ctx, constants.MetadataName, Hostname.(string), r.namespace)
	}
	if r.bmcObject != nil && r.checkIfSystemIsUndergoingReset() {
		l.LogWithFields(ctx).Debugf("Resource %s is already added or undergoing reset so skipping deletion", systemID)
		return
	}
	if r.bmcObject == nil {
		l.LogWithFields(ctx).Debugf("bmc object with systemID %s could not be found hence proceeding to delete", systemID)

		r.commonUtil.BmcDeleteOperation(ctx, aggregationURL, systemID)
	}
}

// bmcAdditionOperation will add bmc in ODIM
func (r PollingReconciler) bmcAdditionOperation(ctx context.Context, restClient restclient.RestClientInterface, sysID string) (bool, string) {
	l.LogWithFields(ctx).Info("adding bmc...")
	bmcUtil := controllers.GetBmcUtils(ctx, r.bmcObject, config.Data.Namespace, &r.commonRec, &restClient, common.GetCommonUtils(restClient), false)
	connMeth := bmcUtil.GetConnectionMethod(r.odimObj)
	if connMeth != "" {
		decryptedPass := utils.DecryptWithPrivateKey(r.ctx, r.bmcObject.Spec.Credentials.Password, *utils.PrivateKey, true) //doBase64Decode = true, because bmc password is encrypted n encoded
		// Prepare Bmc Payload
		body, err := r.prepareBmcPayload(ctx, connMeth, decryptedPass)
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf("error while creating bmc payload %s: ", err.Error()))
			return false, ""
		}

		ok, taskResp := common.GetCommonUtils(restClient).BmcAddition(ctx, r.bmcObject, body)
		if ok && taskResp["Name"].(string) == "Aggregation Source" {
			if task, ok := taskResp["Id"]; ok {
				return true, task.(string)
			}
		}
	}
	return false, ""
}

// prepareBody will prepare the body for add bmc  // TODO: reused, change it
func (r PollingReconciler) prepareBmcPayload(ctx context.Context, connectionMeth, password string) ([]byte, error) {
	connMeth := map[string]string{"@odata.id": connectionMeth}
	link := links{ConnectionMethod: connMeth}
	details := addBmcPayloadBody{HostName: r.bmcObject.Spec.BmcDetails.Address, UserName: r.bmcObject.Spec.Credentials.Username, Password: password, Links: link}
	body, err := json.Marshal(details)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error marshalling add request body for %s BMC: %s", r.bmcObject.Spec.BmcDetails.Address, err.Error()))
		return nil, err
	}
	return body, nil
}

// CheckAndRevertFirmwareVersion function verify the firmwareversion and update it as per the version present in object
func (r PollingReconciler) CheckAndRevertFirmwareVersion(ctx context.Context, bmc v1.Bmc, client restclient.RestClientInterface) {
	var body interface{}
	firmwareObj := r.commonRec.GetFirmwareObject(ctx, constants.MetadataName, bmc.ObjectMeta.Name, bmc.Namespace)
	firmUtil := firmware.GetFirmwareUtils(ctx, firmwareObj, r.commonRec, client, bmc.Namespace)
	if firmwareObj == nil {
		l.LogWithFields(ctx).Info(fmt.Sprintf("firmware object not present %s: ", bmc.Status.FirmwareVersion))
		return
	}
	firmwareVersion := firmUtil.GetFirmwareVersion()
	if bmc.Status.FirmwareVersion != firmwareVersion {
		body = firmware.FirmPayloadWithoutAuth{Image: firmwareObj.Status.ImagePath, Targets: []string{"/redfish/v1/Systems/" + bmc.Status.BmcSystemID}}
		marshalInput, err := json.Marshal(body)
		if err != nil {
			l.LogWithFields(ctx).Error("Failed to marshal firmware input" + err.Error())
			return
		}
		common.MapOfFirmware[bmc.GetName()] = true
		go firmUtil.CreateFirmwareInSystem(marshalInput)
	}
}

// CheckAndRevertBoot is used to check and revert boot details as per the details present in boot object
func (r PollingReconciler) CheckAndRevertBoot(ctx context.Context, bmc v1.Bmc, client restclient.RestClientInterface) {
	bootObj := r.commonRec.GetBootObject(ctx, constants.MetadataName, bmc.GetName(), bmc.Namespace)
	if bootObj == nil {
		l.LogWithFields(ctx).Info(fmt.Sprintf("no boot object present for bmc %s : ", bmc.Name))
		return
	}
	bootUtil := boot.GetBootUtils(ctx, bootObj, r.commonRec, client, common.GetCommonUtils(client), bmc.Namespace)
	sysDetails := common.GetCommonUtils(client).GetBmcSystemDetails(ctx, &bmc)
	bootSetting := bootUtil.GetBootAttributes(sysDetails)
	if !reflect.DeepEqual(&bootObj.Status.Boot, bootSetting) {
		bootObj.Spec.Boot = &v1.BootSetting{}
		bootObj.Spec.Boot.BootOrder = bootObj.Status.Boot.BootOrder
		ok := bootUtil.UpdateBootDetails(ctx, bmc.Status.BmcSystemID, bmc.Name, bootObj, &bmc)
		if ok {
			l.LogWithFields(ctx).Info(fmt.Sprintf("Boot order setting configured for %s BMC", bmc.Name))
		}
	}
}
