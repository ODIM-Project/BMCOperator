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
	"path"
	"regexp"
	"strings"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var resourceReconcilerMap map[string]string = map[string]string{
	"bmc": "/redfish/v1/Systems/[a-zA-Z0-9._-]+[/]*$",
}

// OdimEventMessage contains information of Events and message details including arguments
type OdimEventMessage struct {
	OdataType string      `json:"@odata.type"`
	Name      string      `json:"Name"`
	Context   string      `json:"@odata.context"`
	Events    []OdimEvent `json:"Events"`
}

// OdimEvent contains the details of the event subscribed from PMB
type OdimEvent struct {
	EventType         string `json:"EventType"`
	EventID           string `json:"EventId"`
	Severity          string `json:"Severity"`
	EventTimestamp    string `json:"EventTimestamp"`
	Message           string `json:"Message"`
	MessageID         string `json:"MessageId"`
	OriginOfCondition *Link  `json:"OriginOfCondition,omitempty"`
}

// Link  property shall contain a link to the resource or object that originated the condition that caused the event to be generated
type Link struct {
	Oid string `json:"@odata.id"`
}

// ProcessOdimEvent receives the event from ODIM, validate it
// and trigger appropriate reconciler to sync the changes to server operator
func ProcessOdimEvent(ctx context.Context, client client.Client,
	Scheme *runtime.Scheme, eventName string, event OdimEvent) {

	for resource, regexString := range resourceReconcilerMap {
		regExp := regexp.MustCompile(regexString)
		if ok := regExp.MatchString(event.OriginOfCondition.Oid); ok {
			switch resource {
			case "bmc":
				r, err := GetPollingReconciler(ctx, client, Scheme)
				if err != nil {
					l.LogWithFields(ctx).Warnf("Event can not be processed: %s", err.Error())
					return
				}
				r.processEventsForSystemsResource(ctx, event.MessageID, event.OriginOfCondition.Oid)
			}
		}
	}
}

func (r *PollingReconciler) processEventsForSystemsResource(ctx context.Context, messageID string, originOfCondition string) {
	client, err := restclient.NewRestClient(ctx, r.odimObj, r.commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get rest client for BMC: %s", err.Error())
	}
	r.pollRestClient = client
	r.commonUtil = common.GetCommonUtils(client)
	r.ctx = ctx
	r.namespace = config.Data.Namespace
	l.LogWithFields(ctx).Debugf("Processing events for system resources with messageID %s originOfCondition %s", messageID, originOfCondition)
	switch {
	case strings.Contains(messageID, "ResourceAdded"):
		var bs common.BmcState
		bs.IsAdded = true
		if config.Data.Reconciliation == constants.Accommodate {
			r.AccommodateBMCInfo(ctx, client, originOfCondition, bs)
		} else if config.Data.Reconciliation == constants.Revert {
			r.RevertResourceAdded(ctx, originOfCondition)
		}
	case strings.Contains(messageID, "ResourceRemoved"):
		var bs common.BmcState
		bs.IsDeleted = true
		if config.Data.Reconciliation == constants.Accommodate {
			r.AccommodateBMCInfo(ctx, client, originOfCondition, bs)
		} else if config.Data.Reconciliation == constants.Revert {
			r.RevertResourceRemoved(ctx, originOfCondition)
		}
	case strings.Contains(messageID, "ServerPostDiscoveryComplete"):
		systemID := path.Base(originOfCondition)
		r.bmcObject = r.commonRec.GetBmcObject(ctx, constants.StatusBmcSystemID, systemID, r.namespace)
		biosObj := &infraiov1.BiosSetting{}
		biosUtil := bios.GetBiosUtils(ctx, biosObj, r.commonRec, client, r.namespace)
		volumeObjectsForStorageControllerInOperator := r.commonRec.GetAllVolumeObjectIds(ctx, r.bmcObject, r.namespace)
		volumeObjectsForStorageControllerFromODIM := r.getAllVolumeObjectIdsFromODIM(r.bmcObject, ctx)
		l.LogWithFields(ctx).Debug("Volumes from operator:", volumeObjectsForStorageControllerInOperator)
		l.LogWithFields(ctx).Debug("Volumes from ODIM:", volumeObjectsForStorageControllerFromODIM)
		if config.Data.Reconciliation == constants.Accommodate {
			r.CheckAndAccomodateFirmware(ctx, client)
			r.CheckAndAccommodateBoot(ctx, client)
			if volumeObjectsForStorageControllerInOperator != nil && volumeObjectsForStorageControllerFromODIM != nil {
				r.checkVolumeCreatedAndAccommodate(r.bmcObject, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
				r.checkVolumeDeletedAndAccommodate(r.bmcObject, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
			}
			r.AccommodateBiosInfo(ctx, biosUtil, systemID)
		} else if config.Data.Reconciliation == constants.Revert {
			r.CheckAndRevertFirmwareVersion(ctx, *r.bmcObject, client)
			r.CheckAndRevertBoot(ctx, *r.bmcObject, client)
			r.CheckAndRevertBios(ctx, *r.bmcObject, client)
			if volumeObjectsForStorageControllerInOperator != nil && volumeObjectsForStorageControllerFromODIM != nil {
				r.checkVolumeCreatedAndRevert(r.bmcObject, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
				r.checkVolumeDeletedAndRevert(r.bmcObject, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
			}
		}
	}
}
