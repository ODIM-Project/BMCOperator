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
	"fmt"
	"net/http"
	"strings"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	volume "github.com/ODIM-Project/BMCOperator/controllers/volume"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// -------------------------------ACCOMMODATE/REVERT GO FUNCS-------------------------------------------------
// AccommodateVolumeDetails will accommodate volumes created/deleted as part of Restclient
// ODIM's state is given priority over Operator
// volumeObjectsForStorageControllerInOperator : Existing volume objects in operator
// volumeObjectsForStorageControllerFromODIM : all volume objects present in ODIM
func (pr PollingReconciler) AccommodateVolumeDetails() {
	allBmcObjects := pr.commonRec.GetAllBmcObject(pr.ctx, pr.namespace)
	if allBmcObjects != nil {
		for _, bmc := range *allBmcObjects {
			l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Reconciling volumes for %s bmc", bmc.ObjectMeta.Name))
			volumeObjectsForStorageControllerInOperator := pr.commonRec.GetAllVolumeObjectIds(pr.ctx, &bmc, pr.namespace)
			volumeObjectsForStorageControllerFromODIM := pr.getAllVolumeObjectIdsFromODIM(&bmc, pr.ctx)
			l.LogWithFields(pr.ctx).Debug("Volumes from operator:", volumeObjectsForStorageControllerInOperator)
			l.LogWithFields(pr.ctx).Debug("Volumes from ODIM:", volumeObjectsForStorageControllerFromODIM)
			if volumeObjectsForStorageControllerInOperator != nil && volumeObjectsForStorageControllerFromODIM != nil {
				go pr.checkVolumeCreatedAndAccommodate(&bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
				go pr.checkVolumeDeletedAndAccommodate(&bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
			} else {
				l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Could not successfully retrieve volume IDs for %s Bmc", bmc.ObjectMeta.Name))
				continue
			}
		}
	}
}

// AccommodateVolumeDetails will accommodate volumes created/deleted as part of Restclient
// Operator's state is given priority over ODIM
// volumeObjectsForStorageControllerInOperator : Existing volume objects in operator
// volumeObjectsForStorageControllerFromODIM : all volume objects present in ODIM
func (pr PollingReconciler) RevertVolumeDetails() {
	allBmcObjects := pr.commonRec.GetAllBmcObject(pr.ctx, pr.namespace)
	if allBmcObjects != nil {
		for _, bmc := range *allBmcObjects {
			l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Reconciling volumes for %s bmc", bmc.ObjectMeta.Name))
			volumeObjectsForStorageControllerInOperator := pr.commonRec.GetAllVolumeObjectIds(pr.ctx, &bmc, pr.namespace)
			volumeObjectsForStorageControllerFromODIM := pr.getAllVolumeObjectIdsFromODIM(&bmc, pr.ctx)
			l.LogWithFields(pr.ctx).Debug("Volumes from operator:", volumeObjectsForStorageControllerInOperator)
			l.LogWithFields(pr.ctx).Debug("Volumes from ODIM:", volumeObjectsForStorageControllerFromODIM)
			if volumeObjectsForStorageControllerInOperator != nil && volumeObjectsForStorageControllerFromODIM != nil {
				go pr.checkVolumeCreatedAndRevert(&bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
				go pr.checkVolumeDeletedAndRevert(&bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
			} else {
				l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Could not successfully retrieve volume IDs for %s Bmc", bmc.ObjectMeta.Name))
				continue
			}
		}
	}
}

// ---------------------------------------------ACCOMMODATE USE CASE-------------------------------------------------------------
// checkVolumeCreatedAndAccomodate will create volume objects in operator when new volumes are added to ODIM
// Example:
// operator: {"ArrayController-0":[]string{"1","2","3"},"ArrayController-1":{"5","7"},"ArrayController-2":{"8"}}
// ODIM:     {"ArrayController-0":[]string{"1","2","3","4"},"ArrayController-1":{"5","7"},"ArrayController-2":{"8"}}
// newCreatedVolumes : {"ArrayController-0":[]string{"4"},"ArrayController-2":{"8"}}
// filteredOutVolumesAlreadyInProgress will filter out all volumes from newCreatedVolumes whose object is being created, i.e operation is inprogress
// volume objects for volumes in filteredOutVolumesAlreadyInProgress will be created by operator
func (pr PollingReconciler) checkVolumeCreatedAndAccommodate(bmc *infraiov1.Bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator map[string][]string) {
	// do not change the sequence in which maps are sent to getVolumes
	newCreatedVolumes := pr.getVolumes(volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
	l.LogWithFields(pr.ctx).Debug("Difference in volumes from ODIM & Operator:", newCreatedVolumes)
	for storageController, volumeIds := range newCreatedVolumes {
		filteredOutVolumesAlreadyInProgress := pr.filterOutVolumesInProgress(volumeIds, bmc.ObjectMeta.Name, bmc.Status.BmcSystemID, storageController, "IsGettingCreated")
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Creating volume objects for %s Storage Controller having %s volumes(filtered out inprogress volumes):", storageController, filteredOutVolumesAlreadyInProgress))
		pr.getVolumeDetailsAndCreateVolumeObject(bmc, storageController, filteredOutVolumesAlreadyInProgress)
	}
}

// checkVolumeDeletedAndAccomodate will delete volumes in operator when volumes are deleted in ODIM
// Example:
// operator: {"ArrayController-0":[]string{"1","2","3"},"ArrayController-1":{"5","7"},"ArrayController-2":{"8"}}
// ODIM:     {"ArrayController-0":[]string{"1","2"},"ArrayController-1":{"5"}}
// deletedVolumesPerStorageController : {"ArrayController-0":[]string{"3"},"ArrayController-1":{"7"},"ArrayController-2":{"8"}}
// filteredOutVolumesAlreadyInProgress will filter out all volumes from deletedVolumesPerStorageController whose object is being deleted, i.e operation is inprogress
// volume objects for volumes in filteredOutVolumesAlreadyInProgress will be deleted by operator
func (pr PollingReconciler) checkVolumeDeletedAndAccommodate(bmc *infraiov1.Bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator map[string][]string) {
	// do not change the sequence in which maps are sent to getVolumes
	deletedVolumesPerStorageController := pr.getVolumes(volumeObjectsForStorageControllerInOperator, volumeObjectsForStorageControllerFromODIM)
	l.LogWithFields(pr.ctx).Debug("Difference in volumes from ODIM & Operator:", deletedVolumesPerStorageController)
	for storageController, volumeIds := range deletedVolumesPerStorageController {
		filteredOutVolumesAlreadyInProgress := pr.filterOutVolumesInProgress(volumeIds, bmc.ObjectMeta.Name, bmc.Status.BmcSystemID, storageController, "IsGettingDeleted")
		l.LogWithFields(pr.ctx).Debug(fmt.Sprintf("Deleting volume objects for %s Storage Controller having %s volumes(filtered out inprogress volumes):", storageController, filteredOutVolumesAlreadyInProgress))
		pr.deleteVolumeObjects(bmc, storageController, filteredOutVolumesAlreadyInProgress)
	}
}

// --------------------------------------------REVERT USE CASE-------------------------------------------------------
// checkVolumeCreatedAndRevert will check for new volumes created in ODIM and delete them
// Example:
// operator: {"ArrayController-0":[]string{"1","2","3"},"ArrayController-1":{"5","7"}}
// ODIM:     {"ArrayController-0":[]string{"1","2","3","4"},"ArrayController-1":{"5","7"},"ArrayController-2":{"8"}}
// newCreatedVolumes : {"ArrayController-0":[]string{"4"},"ArrayController-2":{"8"}}
// filteredOutVolumesAlreadyInProgress will filter out all volumes from newCreatedVolumes which is currently being deleted from ODIM, i.e operation is inprogress
// volumes in filteredOutVolumesAlreadyInProgress will be deleted by operator in ODIM
func (pr PollingReconciler) checkVolumeCreatedAndRevert(bmc *infraiov1.Bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator map[string][]string) {
	// do not change the sequence in which maps are sent to getVolumes
	newCreatedVolumes := pr.getVolumes(volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator)
	l.LogWithFields(pr.ctx).Debug("Difference in volumes from ODIM & Operator:", newCreatedVolumes)
	for storageController, volumeIds := range newCreatedVolumes {
		filteredOutVolumesAlreadyInProgress := pr.filterOutVolumesInProgress(volumeIds, bmc.ObjectMeta.Name, bmc.Status.BmcSystemID, storageController, "IsGettingDeleted")
		l.LogWithFields(pr.ctx).Debug(fmt.Sprintf("Deleting volumes from ODIM for %s Storage Controller having %s volumes(filtered out inprogress volumes):", storageController, filteredOutVolumesAlreadyInProgress))
		pr.getVolumeDetailsAndDeleteVolumeInODIM(bmc, storageController, filteredOutVolumesAlreadyInProgress)
	}
}

// checkVolumeDeletedAndRevert will check for volumes deleted in ODIM and create them back
// Example:
// operator: {"ArrayController-0":[]string{"1","2","3"},"ArrayController-1":{"5","7"},"ArrayController-2":{"8"}}
// ODIM:     {"ArrayController-0":[]string{"1","2"},"ArrayController-1":{"5"}}
// deletedVolumesPerStorageController : {"ArrayController-0":[]string{"3"},"ArrayController-1":{"7"},"ArrayController-2":{"8"}}
// filteredOutVolumesAlreadyInProgress will filter out all volumes from deletedVolumesPerStorageController which is being created in ODIM, i.e operation is inprogress
// volumes in filteredOutVolumesAlreadyInProgress will be created by operator in ODIM directly taking details from volume object already present
func (pr PollingReconciler) checkVolumeDeletedAndRevert(bmc *infraiov1.Bmc, volumeObjectsForStorageControllerFromODIM, volumeObjectsForStorageControllerInOperator map[string][]string) {
	// do not change the sequence in which maps are sent to getVolumes
	deletedVolumesPerStorageController := pr.getVolumes(volumeObjectsForStorageControllerInOperator, volumeObjectsForStorageControllerFromODIM)
	l.LogWithFields(pr.ctx).Debug("Difference in volumes from ODIM & Operator:", deletedVolumesPerStorageController)
	for storageController, volumeIds := range deletedVolumesPerStorageController {
		filteredOutVolumesAlreadyInProgress := pr.filterOutVolumesInProgress(volumeIds, bmc.ObjectMeta.Name, bmc.Status.BmcSystemID, storageController, "IsGettingCreated")
		l.LogWithFields(pr.ctx).Debug(fmt.Sprintf("Creating volumes in ODIM for %s Storage Controller having %s volumes(filtered out inprogress volumes):", storageController, filteredOutVolumesAlreadyInProgress))
		pr.getVolumeObjectDetailsAndCreateVolumeInODIM(bmc, storageController, filteredOutVolumesAlreadyInProgress)
	}
}

//------------------------------------------------POLL VOLUME UTILS-------------------------------------------------

// getDifferenceInVolumesBetweenOdimAndOperator will get the difference between volumes in ODIM and Operator
func (pr PollingReconciler) getDifferenceInVolumesBetweenOdimAndOperator(volumeObjectIds1 []string, volumeObjectIds2 []string) (bool, []string) {
	var diffVolumes []string
	for _, odimVolId := range volumeObjectIds2 { // 1 2 3 4 5
		present := false
		for _, operatorVolID := range volumeObjectIds1 { // 1 2 3
			if odimVolId == operatorVolID {
				present = true
				break
			}
		}
		if !present {
			diffVolumes = append(diffVolumes, odimVolId)
		}
	}
	if diffVolumes != nil {
		return true, diffVolumes
	}
	return false, nil
}

// getAllVolumeObjectIdsFromODIM will get all the volume object ids currently present in ODIM
func (pr PollingReconciler) getAllVolumeObjectIdsFromODIM(bmc *infraiov1.Bmc, ctx context.Context) map[string][]string {
	var volumesFromODIM = map[string][]string{}
	storageDetails, _, err := pr.pollRestClient.Get(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/", bmc.Status.BmcSystemID), fmt.Sprintf("Fetching Storage details for %s BMC..", bmc.Spec.BmcDetails.Address))
	if err != nil {
		l.LogWithFields(ctx).Error(err, "Error getting storage details")
		return nil
	}
	if storageDetails["Members"] == nil || len(storageDetails["Members"].([]interface{})) == 0 || storageDetails["Members"].([]interface{}) == nil {
		l.LogWithFields(ctx).Info("No storage controllers present..")
		return map[string][]string{}
	} else {
		arrContrDetails := storageDetails["Members"].([]interface{})
		for _, arrayControllerPath := range arrContrDetails { // arrayControllerPath : "@odata.id": "/redfish/v1/Systems/e19e1a0c-32a9-45e9-83d1-53824085df99.1/Storage/ArrayControllers-0"
			path := arrayControllerPath.(map[string]interface{})
			arrayController := strings.Split(path["@odata.id"].(string), "/") //arrayController : [ redfish v1 Systems e19e1a0c-32a9-45e9-83d1-53824085df99.1 Storage ArrayControllers-0]
			volumeResponse, _, err := pr.pollRestClient.Get(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes", bmc.Status.BmcSystemID, arrayController[len(arrayController)-1]), fmt.Sprintf("Fetching Volume details for %s BMC..", bmc.Spec.BmcDetails.Address))
			if err != nil {
				l.LogWithFields(ctx).Error(err, fmt.Sprintf("Error getting volume details for %s", arrayController[len(arrayController)-1]))
			}
			if volumeResponse["Members"] == nil || len(volumeResponse["Members"].([]interface{})) == 0 || volumeResponse["Members"].([]interface{}) == nil {
				volumesFromODIM[arrayController[len(arrayController)-1]] = []string{}
				continue
			}
			volumeDetails := volumeResponse["Members"].([]interface{})
			var volumeIds []string
			for _, eachVolPath := range volumeDetails {
				path := eachVolPath.(map[string]interface{})
				eachVol := path["@odata.id"].(string) //eachVol : redfish/v1/Systems/e19e1a0c-32a9-45e9-83d1-53824085df99.1/Storage/ArrayControllers-0/Volumes/1
				volumeIds = append(volumeIds, string(eachVol[len(eachVol)-1]))
			}
			volumesFromODIM[arrayController[len(arrayController)-1]] = volumeIds
		}
		return volumesFromODIM
	}
}

// getVolumeDetailsAndCreateVolumeObject will get newly created volume details from ODIM and create new objects for it in operator
func (pr PollingReconciler) getVolumeDetailsAndCreateVolumeObject(bmc *infraiov1.Bmc, storageController string, volumeIds []string) {
	for _, volId := range volumeIds {
		volumeURL := fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes/%s", bmc.Status.BmcSystemID, storageController, volId)
		newVolumeDetails, _, err := pr.pollRestClient.Get(volumeURL, fmt.Sprintf("Fetching newly created Volume %s details", volId))
		if err != nil {
			l.LogWithFields(pr.ctx).Error(err, fmt.Sprintf("Error getting new volume %s details", volId))
		}
		var newVolume = infraiov1.Volume{}
		newVolume.ObjectMeta.Namespace = pr.namespace
		volName := newVolumeDetails["Name"].(string)
		if volName != "" {
			volName = strings.ToLower(strings.ReplaceAll(newVolumeDetails["Name"].(string), " ", ""))
		} else {
			l.LogWithFields(pr.ctx).Info("Volume name is not present, hence using the storageID and volume ID as volume name")
			volName = storageController + "_vol" + volId
		}
		newVolume.ObjectMeta.Name = bmc.ObjectMeta.Name + "." + volName
		newVolume.ObjectMeta.Annotations = map[string]string{}
		newVolume.ObjectMeta.Annotations["odata.id"] = volumeURL
		controllerutil.AddFinalizer(&newVolume, volume.VolFinalizer)
		var drives = []string{}
		var identifier infraiov1.Identifier
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Creating volume object %s...", newVolume.ObjectMeta.Name))
		err = pr.Client.Create(pr.ctx, &newVolume)
		if err != nil {
			l.LogWithFields(pr.ctx).Errorf("Failed to create volume object: %v", err)
			continue
		}
		newVolume.Status.RAIDType = newVolumeDetails["RAIDType"].(string)
		newVolume.Status.StorageControllerID = storageController
		driveLinks := newVolumeDetails["Links"].(map[string]interface{})["Drives"].([]interface{})
		for _, dl := range driveLinks {
			drive := dl.(map[string]interface{})["@odata.id"].(string)
			driveURI := strings.Split(drive, "/")
			driveID := driveURI[len(driveURI)-1]
			drives = append(drives, driveID)
		}
		newVolume.Status.Drives = drives
		newVolume.Status.VolumeID = newVolumeDetails["Id"].(string)
		newVolume.Status.CapacityBytes = fmt.Sprintf("%f", newVolumeDetails["CapacityBytes"].(float64))
		identifierDetails := newVolumeDetails["Identifiers"].([]interface{})
		for _, idenDet := range identifierDetails {
			identifier.DurableName = idenDet.(map[string]interface{})["DurableName"].(string)
			identifier.DurableNameFormat = idenDet.(map[string]interface{})["DurableNameFormat"].(string)
		}
		newVolume.Status.Identifiers = identifier
		newVolume.Status.VolumeName = newVolumeDetails["Name"].(string)
		err = pr.commonRec.GetCommonReconcilerClient().Status().Update(pr.ctx, &newVolume)
		if err != nil {
			l.LogWithFields(pr.ctx).Errorf("Error: Updating Status of %s Volume : %s", newVolume.Status.VolumeName, err.Error())
			continue
		}
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Successfully created volume object %s", newVolume.ObjectMeta.Name))
	}
}

// deleteVolumeObjects will delete all the volumes objects in operator for each storage controller
func (pr PollingReconciler) deleteVolumeObjects(bmc *infraiov1.Bmc, storageController string, volumeIds []string) {
	allVolumeObjects := pr.commonRec.GetAllVolumeObjects(pr.ctx, bmc.ObjectMeta.Name, pr.namespace)
	for _, volObj := range allVolumeObjects {
		if volObj.Status.StorageControllerID == storageController {
			for _, volId := range volumeIds {
				if volObj.Status.VolumeID == volId {
					controllerutil.RemoveFinalizer(volObj, volume.VolFinalizer)
					err := pr.commonRec.GetCommonReconcilerClient().Update(pr.ctx, volObj)
					if err != nil {
						l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error while deleting volume object for volume %s:", volObj.Status.VolumeID) + err.Error())
						continue
					}
					l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Deleting %s volume Object", volObj.ObjectMeta.Name))
					pr.commonRec.DeleteVolumeObject(pr.ctx, volObj)
				}
			}
		}
	}
}

// getVolumeObjectDetailsAndCreateVolumeInODIM will get deleted volume details from existing volume objects and create volumes
func (pr PollingReconciler) getVolumeObjectDetailsAndCreateVolumeInODIM(bmcObj *infraiov1.Bmc, storageController string, volumeIds []string) {
	for _, volId := range volumeIds {
		existingVolObj := pr.commonRec.GetVolumeObjectByVolumeID(pr.ctx, volId, pr.namespace)
		pr.volUtil.UpdateVolumeObject(existingVolObj)
		// setting updateVolumeObject = false because we are not creating new volume object ,since we already have an existing one.
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Creating %s volume in ODIM", existingVolObj.ObjectMeta.Name))
		isCreated, err := pr.volUtil.CreateVolume(bmcObj, existingVolObj.Status.VolumeName, types.NamespacedName{Namespace: pr.namespace}, false, pr.commonRec)
		if err != nil {
			l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error while creating volume for %s:", existingVolObj.Name) + err.Error())
			continue
		}
		//reset here
		bmcObj.Spec.BmcDetails.ResetType = "ForceRestart"
		pr.bmcUtil.UpdateBmcObject(bmcObj)
		resetdone := pr.bmcUtil.ResetSystem(false, false) // setting updateBmcDependents = false, since we do not want those actions to take place, we just need to reset for our volume to be created again
		if resetdone {
			if isCreated {
				newVolumeID, err := pr.getNewVolumeIDForObject(bmcObj, storageController, existingVolObj)
				if err != nil || newVolumeID == "" {
					l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error while getting new volume ID for exsisting volume object %s:", existingVolObj.Name) + err.Error())
					continue
				}
				l.LogWithFields(pr.ctx).Debug(fmt.Sprintf("New volume ID for exsisting object %s is : %s", existingVolObj.ObjectMeta.Name, newVolumeID))
				existingVolObj.Status.VolumeID = newVolumeID
				err = pr.commonRec.GetCommonReconcilerClient().Status().Update(pr.ctx, existingVolObj)
				if err != nil {
					l.LogWithFields(pr.ctx).Errorf("Error: Updating Status of %s Volume : %s", existingVolObj.Status.VolumeName, err.Error())
				}
				l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Updated volume %s in ODIM successfully", existingVolObj.ObjectMeta.Name))
			} else {
				l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Could not update volume %s in ODIM", existingVolObj.ObjectMeta.Name))
			}
		} else {
			l.LogWithFields(pr.ctx).Info("Reset operation not successful, volume is not visible on ODIM, hence unable to update new volume ID")
		}
	}
}

// getNewVolumeIDForObject will get the new VolumeID for the volume object whose Volume was deleted from ODIM, and had to be added back as part of Revert use case
func (pr PollingReconciler) getNewVolumeIDForObject(bmcObj *infraiov1.Bmc, storageController string, volObj *infraiov1.Volume) (string, error) {
	getAllVolumesResp, sCode, err := pr.pollRestClient.Get(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes", bmcObj.Status.BmcSystemID, storageController), "Fetching all volumes..")
	if err != nil {
		l.LogWithFields(pr.ctx).Error("error while fetching all volumes:" + err.Error())
		return "", err
	} else if sCode != http.StatusOK {
		l.LogWithFields(pr.ctx).Info("Failed to get all volumes, try again.")
		return "", nil
	}
	volumesList := getAllVolumesResp["Members"].([]interface{})
	for _, vol := range volumesList {
		volURL := vol.(map[string]interface{})["@odata.id"].(string)
		getEachVolResp, sCode, err := pr.pollRestClient.Get(volURL, fmt.Sprintf("Fetching %s volume details..", volURL[len(volURL)-1:]))
		if err != nil {
			l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error while fetching %s volume details:", volURL[len(volURL)-1:]) + err.Error())
			return "", err
		} else if sCode != http.StatusOK {
			l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Failed to get %s volume details, try again.", volURL[len(volURL)-1:]))
			return "", nil
		} else {
			//getting all drives in the volume
			drives := []string{}
			driveLinks := getEachVolResp["Links"].(map[string]interface{})["Drives"].([]interface{})
			for _, dl := range driveLinks {
				drive := dl.(map[string]interface{})["@odata.id"].(string)
				driveURI := strings.Split(drive, "/")
				driveID := driveURI[len(driveURI)-1]
				drives = append(drives, driveID)
			}
			if getEachVolResp["RAIDType"] == volObj.Status.RAIDType && utils.CompareArray(drives, volObj.Status.Drives) {
				return getEachVolResp["Id"].(string), nil
			}
		}
	}
	return "", nil
}

// getVolumeDetailsAndDeleteVolumeInODIM will delete the volumes for each storage controller on ODIM directly
func (pr PollingReconciler) getVolumeDetailsAndDeleteVolumeInODIM(bmcObj *infraiov1.Bmc, storageController string, volumeIds []string) {
	var volObj = infraiov1.Volume{}
	for _, volID := range volumeIds {
		volObj.Status.VolumeID = volID
		volObj.Status.VolumeName = volID // setting volumeName same as volumeID as could not get name, because we are directly using volumeids to get the difference
		volObj.Status.StorageControllerID = storageController
		pr.volUtil.UpdateVolumeObject(&volObj)
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Deleting %s volume in ODIM...", volID))
		pr.volUtil.DeleteVolume(bmcObj)
	}
}

// getVolumes will get the difference between volumeObjectsMap1 and volumeObjectsMap2
// where volumeObjectsMap1 can be either Operator/ODIM's volumes
// and volumeObjectsMap2 can be either Operator/ODIM's volumes
func (pr PollingReconciler) getVolumes(volumeObjectsMap1, volumeObjectsMap2 map[string][]string) map[string][]string {
	var diffVolumesPerStorageController = map[string][]string{} // ArrayController: []string{1,2,...}
	for storageController, volumeIds := range volumeObjectsMap1 {
		if volumeObjectsMap2[storageController] != nil {
			isDifferencePresent, diffVolumes := pr.getDifferenceInVolumesBetweenOdimAndOperator(volumeObjectsMap2[storageController], volumeIds)
			if isDifferencePresent {
				diffVolumesPerStorageController[storageController] = diffVolumes
			}
		} else {
			diffVolumesPerStorageController[storageController] = volumeIds
		}
	}
	return diffVolumesPerStorageController
}

// filterOutVolumesInProgress will filter out all the volumes whose next set of operations based on Accommodate/Revert are in progress, from the difference of volumes in Operator & ODIM
func (pr PollingReconciler) filterOutVolumesInProgress(volumeIds []string, bmcName, bmcSystemID, storageController, operation string) []string {
	filteredVolumes := []string{}
	for _, volID := range volumeIds {
		if operation == "IsGettingDeleted" {
			bmcStorageVolume := common.BmcStorageControllerVolumesStatus{Bmc: bmcName, StorageController: storageController, VolumeID: volID}
			if !common.TrackVolumeStatus[bmcStorageVolume].IsGettingDeleted {
				filteredVolumes = append(filteredVolumes, volID)
			}

		} else if operation == "IsGettingCreated" {
			volumeObjName := bmcName + "." + pr.getVolumeName(bmcSystemID, storageController, volID)
			bmcStorageVolume := common.BmcStorageControllerVolumesStatus{Bmc: bmcName, StorageController: storageController, VolumeID: volumeObjName}
			if volumeObjName != "" && !common.TrackVolumeStatus[bmcStorageVolume].IsGettingCreated {
				filteredVolumes = append(filteredVolumes, volID)
			}
		}
	}
	return filteredVolumes
}

// getVolumeName will return the volume name as present in ODIM based on Volume ID
func (pr PollingReconciler) getVolumeName(bmcSystemID, storageController, volID string) string {
	volResp, sCode, err := pr.pollRestClient.Get(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes/%s", bmcSystemID, storageController, volID), "Getting volume details..")
	if err != nil {
		l.LogWithFields(pr.ctx).Error(fmt.Sprintf("error while getting %s volume details:", volID) + err.Error())
		return ""
	}
	if sCode != http.StatusOK {
		l.LogWithFields(pr.ctx).Info(fmt.Sprintf("Could not retrieve volume %s details..", volID))
		return ""
	}
	return volResp["Name"].(string)
}
