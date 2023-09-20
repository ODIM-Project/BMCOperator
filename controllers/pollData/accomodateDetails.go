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
	"path"
	"reflect"
	"strings"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"

	controllers "github.com/ODIM-Project/BMCOperator/controllers/bmc"
	boot "github.com/ODIM-Project/BMCOperator/controllers/boot"

	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	firmware "github.com/ODIM-Project/BMCOperator/controllers/firmware"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AccommodatePollingDetails will get bmc details and update information accordingly
func (r PollingReconciler) AccommodatePollingDetails(ctx context.Context, restClient restclient.RestClientInterface, mapOfBmc map[string]common.BmcState) {
	for urlpath, bmcState := range mapOfBmc {
		l.LogWithFields(r.ctx).Debugf("Accommodating for %s bmc", urlpath)
		r.AccommodateBMCInfo(ctx, restClient, urlpath, bmcState)
	}
}

// AccommodateBMCInfo will accommodate information for a single BMC
func (r PollingReconciler) AccommodateBMCInfo(ctx context.Context, restClient restclient.RestClientInterface,
	urlpath string, bmcState common.BmcState) {
	var objCreated bool
	// get the system ID from URL path and replace it with aggregation URL to get hostName
	systemID := path.Base(urlpath)
	aggregationURL := strings.Replace(urlpath, "/Systems/", "/AggregationService/AggregationSources/", 1)

	if bmcState.IsAdded || bmcState.IsDeleted {
		r.bmcObject = r.commonRec.GetBmcObject(ctx, constants.StatusBmcSystemID, systemID, r.namespace)
		l.LogWithFields(r.ctx).Debug("bmc object:", r.bmcObject)
		if r.bmcObject != nil {
			if r.checkIfSystemIsUndergoingReset() {
				return
			}

			// TODO: move below functions to different file
			if bmcState.IsObjCreated {
				r.CheckAndAccomodateFirmware(ctx, restClient)
				r.CheckAndAccommodateBoot(ctx, restClient)
			}
			bmcState := common.MapOfBmc[urlpath]
			bmcState.IsObjCreated = true
			common.MapOfBmc[urlpath] = bmcState
			objCreated = true
		} else if r.bmcObject == nil && bmcState.IsDeleted {
			delete(common.MapOfBmc, urlpath)
		}
	}

	// Delete bmc object
	if bmcState.IsDeleted && objCreated {
		l.LogWithFields(ctx).Info("Remove Bmc object: ", r.bmcObject.Name)
		err := r.Client.Delete(ctx, r.bmcObject)
		if err != nil {
			l.LogWithFields(ctx).Info("Object not deleted", err)
			return
		}
		delete(common.MapOfBmc, urlpath)
		return
	}

	//create BMC object
	if !bmcState.IsObjCreated && !objCreated && r.bmcObject == nil && bmcState.IsAdded && !bmcState.IsDeleted {
		l.LogWithFields(ctx).Info("Creating Bmc object")
		res, _, err := restClient.Get(aggregationURL, "Getting response on Aggregation Sources")
		l.LogWithFields(r.ctx).Debugf("Response from GET on %s:%v", aggregationURL, res)
		if err != nil {
			l.LogWithFields(ctx).Errorf("Failed to get Aggregation Sources details")
			return
		}
		if hostName, ok := res["HostName"].(string); ok {
			links := res["Links"].(map[string]interface{})
			connMethodOdata := links["ConnectionMethod"].(map[string]interface{})
			connMethodID := connMethodOdata["@odata.id"].(string)
			connecRes, _, err := restClient.Get(connMethodID, "Getting response on Connection Method")
			l.LogWithFields(r.ctx).Debugf("Response from GET on %s:%v", connMethodID, connecRes)
			if err != nil {
				l.LogWithFields(ctx).Errorf("Failed to get Connection Method details")
				return
			}
			if connMethod, ok := connecRes["ConnectionMethodVariant"]; ok {
				obj := r.createBmcObject(r.Client, "Bmc", hostName, systemID, res["UserName"].(string), connMethod.(string))
				err = r.Client.Create(ctx, obj)
				if err != nil {
					l.LogWithFields(ctx).Errorf("Failed to create object: %v", err)
					return
				}
				common.MapOfBmc[urlpath] = common.BmcState{IsAdded: false, IsDeleted: false, IsObjCreated: true}
			}
		}
	}
	l.LogWithFields(r.ctx).Debug("MapOfBmc in Accommodate:", common.MapOfBmc)
}

// createBmcObject used for creating unstructured bmc object
func (r PollingReconciler) createBmcObject(c client.Client, kind, name, systemID, username, ConnMethVariant string) *infraiov1.Bmc {
	bmcObj := &infraiov1.Bmc{}
	bmcObj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infra.io.odimra",
		Kind:    kind,
		Version: "v1",
	})
	bmcObj.SetName(name)
	if r.odimObj != nil {
		bmcObj.SetNamespace(r.odimObj.Namespace)
	}
	bmcObj.Spec.BmcDetails.ConnMethVariant = ConnMethVariant
	bmcObj.Status.BmcSystemID = systemID
	bmcObj.Spec.BmcDetails.Address = name
	bmcObj.Spec.Credentials.Username = username
	bmcObj.Labels = map[string]string{"systemId": systemID, "name": name}
	resourceBytes, err := json.Marshal(bmcObj)
	if err != nil {
		return nil
	}
	annotations := bmcObj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(resourceBytes)
	annotations["odata.id"] = "/redfish/v1/Systems/" + systemID
	bmcObj.SetAnnotations(annotations)

	return bmcObj
}

// CheckAndAccomodateFirmware is used to check, update firmware version in bmc & firmware object
func (r PollingReconciler) CheckAndAccomodateFirmware(ctx context.Context, client restclient.RestClientInterface) {
	bmcUtil := controllers.GetBmcUtils(ctx, r.bmcObject, r.bmcObject.Namespace, &r.commonRec, &client, common.GetCommonUtils(client), false)
	mgrDetails := bmcUtil.GetManagerDetails()
	if mgrDetails != nil {
		if firmwareVersion, ok := mgrDetails["FirmwareVersion"].(string); ok {
			firmwareObj := r.commonRec.GetFirmwareObject(ctx, constants.MetadataName, r.bmcObject.ObjectMeta.Name, r.bmcObject.Namespace)
			firmUtil := firmware.GetFirmwareUtils(ctx, firmwareObj, r.commonRec, client, r.bmcObject.Namespace)
			if firmwareVersion != r.bmcObject.Status.FirmwareVersion {
				// Update firware version with latest value in bmc object and firmware object
				if firmwareObj != nil {
					firmUtil.UpdateBmcObjectWithFirmwareVersionAndSchema(r.bmcObject, firmwareVersion, "")
					firmUtil.UpdateFirmwareLabels(firmwareVersion)
					imagePath := getImagePath(firmwareVersion, firmwareObj.Status.ImagePath)
					firmUtil.UpdateFirmwareStatus("Success", firmwareVersion, imagePath)
				}
			}
		}
	}
}

// CheckAndAccommodateBoot is used to check, and update the boot object as per the changes made on bmc
func (r PollingReconciler) CheckAndAccommodateBoot(ctx context.Context, client restclient.RestClientInterface) {
	bootObj := r.commonRec.GetBootObject(ctx, constants.MetadataName, r.bmcObject.GetName(), r.bmcObject.Namespace)
	bootUtil := boot.GetBootUtils(ctx, bootObj, r.commonRec, client, common.GetCommonUtils(client), r.bmcObject.Namespace)
	sysDetails := common.GetCommonUtils(client).GetBmcSystemDetails(ctx, r.bmcObject)
	bootSetting := bootUtil.GetBootAttributes(sysDetails)
	if !reflect.DeepEqual(bootSetting, &bootObj.Status.Boot) {
		bootObj.Status.Boot = *bootSetting
		err := r.commonRec.GetCommonReconcilerClient().Status().Update(ctx, bootObj)
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf("Error while updating boot order setting object for %s BMC: %s", bootObj.Name, err.Error()))
		}
	}
}

// getImagePath is used to get the image path for saving it in object
func getImagePath(firmwareVersion, path string) string {
	split := strings.SplitN(firmwareVersion, "v", 2)
	lowerCaseVersion := strings.ToLower(split[0])
	lowerCaseVersion = strings.Replace(lowerCaseVersion, " ", "", -1)
	version := strings.Replace(split[1], ".", "", -1)
	finalpath := strings.Replace(lowerCaseVersion+" "+version, " ", "_", -1)
	return replacePath(path, finalpath+".bin")
}

// replacePath is used to get final path to be saved in object
func replacePath(path string, replacement string) string {
	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return replacement
	}
	replacedPath := path[:lastSlashIndex+1] + replacement
	return replacedPath
}
