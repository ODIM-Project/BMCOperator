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
	"net/url"
	"sync"
	"time"

	v1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	bmc "github.com/ODIM-Project/BMCOperator/controllers/bmc"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	volume "github.com/ODIM-Project/BMCOperator/controllers/volume"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// PollDetails is used to get data a some time interval
func PollDetails(mgr manager.Manager) {
	ctx := context.Background()
	transactionID := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionID.String(), constants.BmcOperator, constants.PollingActionID, constants.PollingActionName, podName)
	r := PollingReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()} //TODO: getPollingUtils
	// Set time ticker
	interval := time.Duration(config.TickerTime) * time.Hour
	if config.TickerTime == 0 {
		interval = time.Duration(constants.DefaultTimeTicker) * time.Hour
	}
	config.Ticker = time.NewTicker(interval)
	defer config.Ticker.Stop()
	var wg sync.WaitGroup
	for range config.Ticker.C {
		l.LogWithFields(ctx).Info("reconciling....")

		if config.Data.Reconciliation != "" || config.Data.EventSubReconciliation != "" {
			if r.odimObj == nil {
				r.commonRec = utils.GetCommonReconciler(r.Client, r.Scheme)
				r.odimObj = r.commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", config.Data.Namespace)
				if r.odimObj == nil {
					continue
				}
			}
			client, err := restclient.NewRestClient(ctx, r.odimObj, r.commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
			if err != nil {
				l.LogWithFields(ctx).Errorf("Failed to get rest client for BMC: %s", err.Error())
				continue
			}
			res, _, err := client.Get("/redfish/v1/Systems", "Getting response on systems")
			if err != nil {
				l.LogWithFields(ctx).Errorf("Failed to get system details")
				continue
			}
			//TODO: getutils will handle it
			r.ctx = ctx
			r.namespace = config.Data.Namespace
			r.pollRestClient = client
			r.commonUtil = common.GetCommonUtils(client)
			r.bmcUtil = bmc.GetBmcUtils(ctx, nil, r.namespace, &r.commonRec, &r.pollRestClient, r.commonUtil, true)
			r.volUtil = volume.GetVolumeUtils(ctx, client, r.commonRec, &v1.Volume{}, r.namespace)
			allBmcObjects := r.commonRec.GetAllBmcObject(r.ctx, r.namespace)
			mapOfSystems := make(map[string]bool)
			// Iterate to the systems
			for _, mem := range res["Members"].([]interface{}) {
				members := mem.(map[string]interface{})
				member, err := url.Parse(members["@odata.id"].(string))
				if err != nil {
					l.LogWithFields(ctx).Errorf("error parsing URL: %v", err)
					continue
				}
				/* If new member is added then set value for IsAdded as true
				   mapOfSystems = current added bmcs in ODIM'
				   MapofBmc = bmc objects in operator */
				mapOfSystems[member.Path] = true
				if _, ok := common.MapOfBmc[member.Path]; !ok {
					bmcState := common.MapOfBmc[member.Path]
					bmcState.IsAdded = true
					common.MapOfBmc[member.Path] = bmcState
				}
			}
			if config.Data.Reconciliation == constants.Revert {
				wg.Add(1)
				go func() {
					defer wg.Done()
					r.RevertPollingDetails(ctx, client, mapOfSystems)
					r.RevertBiosDetails(ctx, client)
					r.RevertVolumeDetails()
					if allBmcObjects != nil {
						for _, bmc := range *allBmcObjects {
							r.bmcObject = &bmc
							if !r.checkIfSystemIsUndergoingReset() {
								r.RevertState()
							}
						}
					}
				}()
				wg.Wait()

			}
			if config.Data.Reconciliation == constants.Accommodate {
				// check if system is present, if not then set state IsDeleted to true
				if len(mapOfSystems) != len(common.MapOfBmc) {
					for urlKey := range common.MapOfBmc {
						if _, ok := mapOfSystems[urlKey]; !ok {
							bmcState := common.MapOfBmc[urlKey]
							bmcState.IsDeleted = true
							common.MapOfBmc[urlKey] = bmcState
						}
					}
				}
				l.LogWithFields(ctx).Info("map containing members: ", common.MapOfBmc)
				r.AccommodatePollingDetails(ctx, client, common.MapOfBmc)
				r.AccommodateBiosDetails(ctx, client, mapOfSystems)
				r.AccommodateVolumeDetails()
				if allBmcObjects != nil {
					for _, bmc := range *allBmcObjects {
						r.bmcObject = &bmc
						if !r.checkIfSystemIsUndergoingReset() {
							r.AccommodateState()
						}
					}
				}
			}

			defaultEventsubscriptionDestination := r.odimObj.Spec.EventListenerHost + "/OdimEvents"
			if config.Data.EventSubReconciliation == constants.Revert {
				wg.Add(1)
				go func() {
					defer wg.Done()
					r.RevertEventSubscriptionDetails(ctx, client, defaultEventsubscriptionDestination)
				}()
				wg.Wait()
			}
			if config.Data.EventSubReconciliation == constants.Accommodate {
				r.AccommodateEventSubscriptionDetails(ctx, client, defaultEventsubscriptionDestination)
			}
		}
	}
}

func (pr PollingReconciler) checkIfSystemIsUndergoingReset() bool {
	systemDetails := pr.commonUtil.GetBmcSystemDetails(pr.ctx, pr.bmcObject)
	if statusOfBMC, ok := systemDetails["Status"]; ok {
		if stateOfBMC, ok := statusOfBMC.(map[string]interface{})["State"], ok; ok {
			//checking if reset is triggered from operator or ODIM directly
			if (pr.bmcObject.Status.SystemReset != "" && pr.bmcObject.Status.SystemReset != "Done") || stateOfBMC.(string) == "Starting" {
				return true
			}
		} else {
			l.LogWithFields(pr.ctx).Info("Could not access the state of BMC")
		}
		l.LogWithFields(pr.ctx).Info("Could not access the status of BMC")
	}
	return false
}
