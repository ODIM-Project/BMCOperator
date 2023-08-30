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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	v1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	l "github.com/ODIM-Project/BMCOperator/logs"
)

// Created common folder to avoid import cycle issue
// Functions of bmc that are used in other folders are places here for access

// GetSystemDetails used to get system response
func (bu *CommonUtils) GetBmcSystemDetails(ctx context.Context, bmcObj *infraiov1.Bmc) map[string]interface{} {
	uri := "/redfish/v1/Systems/" + bmcObj.Status.BmcSystemID
	resp, sCode, err := bu.restClient.Get(uri, fmt.Sprintf("Fetching system details for %s BMC", bmcObj.Spec.BmcDetails.Address))
	if sCode == http.StatusOK {
		return resp
	}
	if sCode == http.StatusNotFound {
		l.LogWithFields(ctx).Warnf("No systems found with %s", bmcObj.Status.BmcSystemID)
		return nil
	}
	l.LogWithFields(ctx).Error(fmt.Sprintf("Failed getting system details for %s BMC: Got %d from odim: %s", bmcObj.Spec.BmcDetails.Address, sCode, err.Error()))
	return nil
}

func (bu *CommonUtils) MoniteringTaskmon(headerInfo http.Header, ctx context.Context, operation, resourceName string) (bool, map[string]interface{}) {
	retries := 0
	for retries < constants.RetryCount {
		retries = retries + 1
		time.Sleep(time.Duration(constants.SleepTime) * time.Second)
		taskResp, sc, err := bu.restClient.Get(headerInfo["Location"][0], "Monitoring taskmon...")
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf(TaskmonLogsTable["Err:"+operation], resourceName), err.Error())
		} else {
			l.LogWithFields(ctx).Info(fmt.Sprintf(TaskmonLogsTable[strconv.Itoa(sc)+":"+operation], resourceName))
			if sc == http.StatusAccepted { // taskmon in progress
				continue
			} else if sc == http.StatusOK { // Reset success : 200
				return true, taskResp
			} else if sc == http.StatusCreated && operation == ADDBMC { // Reset success : 201
				return true, taskResp
			} else if sc == http.StatusNotFound && operation == DELETEBMC { // Resource not present for deletion
				return true, taskResp
			} else if sc == http.StatusCreated && operation == EVENTSUBSCRIPTION {
				return true, taskResp
			} else if sc == http.StatusNoContent && operation == DELETEBMC { // Delete success : 204
				return true, taskResp
			} else if sc == http.StatusNotFound { // Resource not present : 404
				return false, taskResp
			} else if sc == http.StatusBadRequest { // Bad Request : 400
				return false, taskResp
			} else if sc == http.StatusConflict { //conflict : 409
				return false, taskResp
			} else {
				return false, taskResp
			}
		}
	}
	return false, nil
}

func GetCommonUtils(restClient restclient.RestClientInterface) CommonInterface {
	return &CommonUtils{restClient: restClient}
}

// BmcAddition is used to add bmc in ODIM
func (bu *CommonUtils) BmcAddition(ctx context.Context, bmcObject *v1.Bmc, body []byte, restClient restclient.RestClientInterface) (bool, map[string]interface{}) {
	uri := "/redfish/v1/AggregationService/AggregationSources/"
	resp, err := restClient.Post(uri, "Posting add bmc payload..", body)
	if err != nil {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Error adding %s BMC", bmcObject.Spec.BmcDetails.Address))
		return false, nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		l.LogWithFields(ctx).Info(fmt.Sprintf("password is not authorised for %s BMC", bmcObject.Spec.BmcDetails.Address))
		return false, nil
	}
	if resp.StatusCode == http.StatusAccepted {
		return bu.MoniteringTaskmon(resp.Header, ctx, ADDBMC, bmcObject.ObjectMeta.Name)
	}
	return false, nil
}

// BmcDeleteOperation will delete the bmc from ODIM
func (bu *CommonUtils) BmcDeleteOperation(ctx context.Context, aggregationURL string, restClient restclient.RestClientInterface, resourceName string) bool {
	delResp, err := restClient.Delete(aggregationURL, "Deleting BMC..")
	if err != nil {
		l.LogWithFields(ctx).Warnf("could not delete bmc, error occurred %v", err)
		return false
	}
	if err == nil && delResp.StatusCode == http.StatusAccepted {
		done, _ := bu.MoniteringTaskmon(delResp.Header, ctx, DELETEBMC, resourceName)
		if done {
			return true
		}
	}
	return false
}

func BiosAttributeUpdation(ctx context.Context, body map[string]interface{}, systemID string, client restclient.RestClientInterface) bool {
	bios_settings := map[string]map[string]interface{}{"Attributes": body}
	biosBody, err := json.Marshal(bios_settings)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to marshal Bios Payload: %s", err.Error())
		return false
	}
	patchResponse, err := client.Patch("/redfish/v1/Systems/"+systemID+"/Bios/Settings/", fmt.Sprintf("Patching bios payload for %s BMC", systemID), biosBody)
	if err == nil && patchResponse.StatusCode == http.StatusAccepted {
		headerInfo := patchResponse.Header
		//monitering taskmon
		retries := 0
		for retries < constants.RetryCount {
			retries = retries + 1
			time.Sleep(time.Duration(constants.SleepTime) * time.Second)
			_, statusCode, err := client.Get(headerInfo["Location"][0], "Monitoring taskmon for bios patch..")
			if err == nil && statusCode == http.StatusOK {
				l.LogWithFields(ctx).Info(fmt.Sprintf("Reset the system: %s", systemID))
				return true
			}
		}
	}
	return false
}
