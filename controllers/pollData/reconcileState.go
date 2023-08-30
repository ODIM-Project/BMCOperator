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
	"encoding/json"
	"fmt"
	"net/http"

	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	l "github.com/ODIM-Project/BMCOperator/logs"
)

func (pr *PollingReconciler) AccommodateState() {
	powerStateOfBMC, powerStateInBMCObject := pr.getPowerStateOnBMCAndObject()
	l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Configured power state for %s BMC in object: `%s`", pr.bmcObject.Spec.BmcDetails.Address, powerStateInBMCObject))
	if powerStateOfBMC != powerStateInBMCObject {
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Accommodating %s power state for %s BMC", powerStateOfBMC, pr.bmcObject.Spec.BmcDetails.Address))
		pr.bmcObject.Status.PowerState = powerStateOfBMC
		err := pr.commonRec.GetCommonReconcilerClient().Status().Update(pr.ctx, pr.bmcObject)
		if err != nil {
			l.LogWithFields(pr.ctx).Errorf("Failed to update %s BMC: %s", pr.bmcObject.ObjectMeta.Name, err.Error())
		}
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Successfully accommodated power state for %s BMC to %s", pr.bmcObject.ObjectMeta.Name, powerStateOfBMC))
	}
}

func (pr *PollingReconciler) RevertState() {
	var request common.ComputerSystemReset
	powerStateOfBMC, powerStateInBMCObject := pr.getPowerStateOnBMCAndObject()
	l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Configured power state for %s BMC in object: `%s`", pr.bmcObject.Spec.BmcDetails.Address, powerStateInBMCObject))
	if powerStateOfBMC != powerStateInBMCObject {
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Reverting %s power state for %s BMC", powerStateInBMCObject, pr.bmcObject.Spec.BmcDetails.Address))
		resetPowerStateURI := fmt.Sprintf("/redfish/v1/Systems/%s/Actions/ComputerSystem.Reset", pr.bmcObject.Status.BmcSystemID)
		if powerStateInBMCObject == "Off" {
			request = common.ComputerSystemReset{ResetType: "ForceOff"}
		} else {
			request = common.ComputerSystemReset{ResetType: powerStateInBMCObject}
		}
		resetReq, err := json.Marshal(request)
		if err != nil {
			l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error marshalling reset request for %s BMC", pr.bmcObject.ObjectMeta.Name), err)
		}
		resp, err := pr.pollRestClient.Post(resetPowerStateURI, fmt.Sprintf("Posting reset request to %s BMC", pr.bmcObject.ObjectMeta.Name), resetReq)
		if err != nil {
			l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error while posting reset request to %s BMC", pr.bmcObject.ObjectMeta.Name), err)
			return
		}
		if resp.StatusCode == http.StatusAccepted {
			done, _ := pr.commonUtil.MoniteringTaskmon(resp.Header, pr.ctx, common.RESETBMC, pr.bmcObject.ObjectMeta.Name)
			if done {
				l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Successfully reverted power state for %s BMC to %s", pr.bmcObject.ObjectMeta.Name, powerStateInBMCObject))
			}
		} else {
			l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Unable to revert power state for %s BMC", pr.bmcObject.ObjectMeta.Name))
		}
	}
}

func (pr *PollingReconciler) getPowerStateOnBMCAndObject() (string, string) {
	var powerStateOfBMC string
	sysRes := pr.commonUtil.GetBmcSystemDetails(pr.ctx, pr.bmcObject)
	if val, ok := sysRes["PowerState"]; ok {
		powerStateOfBMC = val.(string)
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Current power state for %s BMC: `%s`", pr.bmcObject.Spec.BmcDetails.Address, powerStateOfBMC))
	}
	powerStateInBMCObject := pr.bmcObject.Status.PowerState
	return powerStateOfBMC, powerStateInBMCObject
}
