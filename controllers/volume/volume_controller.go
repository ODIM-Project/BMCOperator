//(C) Copyright [2022] Hewlett Packard Enterprise Development LP
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
	"os"
	"reflect"
	"strings"
	"time"

	Error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var podName = os.Getenv("POD_NAME")

// VolumeReconciler reconciles a Volume object
type VolumeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infra.io.odimra,resources=volumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=volumes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=volumes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Volume object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *VolumeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//common reconciler ddeclaration
	transactionId := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionId.String(), constants.BmcOperator, constants.VolumeActionID, constants.VolumeActionName, podName)
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	//create rest client
	odimObj := commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", req.Namespace)
	volRestClient, err := restclient.NewRestClient(ctx, odimObj, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get rest client for Volume: %s", err.Error())
		return ctrl.Result{}, err
	}
	volObj := &infraiov1.Volume{}
	// Fetch the Volume instance.
	err = r.Get(ctx, req.NamespacedName, volObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	//get volumeUtil object
	volUtil := GetVolumeUtils(ctx, volRestClient, commonRec, volObj, req.Namespace)
	// get bmcname and volume name
	var bmcName, dispName string
	// ex: 10.24.0.14.volume1
	strArr := strings.Split(volObj.ObjectMeta.Name, ".")
	dispName = strArr[len(strArr)-1]                    // last name is volume name : volume1
	bmcName = strings.Join(strArr[:len(strArr)-1], ".") //first name is bmc name : 10.24.0.14
	bmcObj := commonRec.GetBmcObject(ctx, constants.MetadataName, bmcName, req.Namespace)
	if bmcObj == nil {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Could not find %s BMC object, hence cannot create volume", bmcName))
		err = r.Delete(ctx, volObj)
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Deleting %s Volume object: %s", dispName, err.Error()))
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	//Checking for deletion
	isVolumeMarkedToBeDeleted := volObj.GetDeletionTimestamp() != nil
	if isVolumeMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(volObj, VolFinalizer) {
			isVolumeDeleted := volUtil.DeleteVolume(bmcObj)
			if isVolumeDeleted {
				controllerutil.RemoveFinalizer(volObj, VolFinalizer)
				err := r.Update(ctx, volObj)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else {
				l.LogWithFields(ctx).Info(fmt.Sprintf("Volume %s not deleted! Retrying..", volObj.Status.VolumeName))
				return ctrl.Result{Requeue: true}, nil // Calling Reconcile again for delete to execute
			}
		}
		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(volObj.Spec, infraiov1.VolumeSpec{}) {
		return ctrl.Result{}, nil
	}
	if volObj.Spec.StorageControllerID != "" && volObj.Spec.RAIDType != "" && len(volObj.Spec.Drives) != 0 {
		// Add finalizer for this CR
		if !controllerutil.ContainsFinalizer(volObj, VolFinalizer) {
			controllerutil.AddFinalizer(volObj, VolFinalizer)
			err := r.Client.Update(ctx, volObj)
			if err != nil {
				l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating %s Volume object: %s", volObj.Status.VolumeName, err.Error()))
				return ctrl.Result{}, err
			}
			time.Sleep(time.Duration(constants.SleepTime) * time.Second)
		}
		//create volume
		volCreated, volErr := volUtil.CreateVolume(bmcObj, dispName, req.NamespacedName, true, commonRec) // setting updateVolumeObject = true since we are creating volume in ODIM and creating object first time
		if !volCreated {
			return ctrl.Result{}, volErr
		}
	} else {
		l.LogWithFields(ctx).Info("Please enter all the details required for volume creation")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.Volume{}).
		WithEventFilter(utils.IgnoreStatusUpdate()).
		Complete(r)
}

// CreateVolume will create volume in ODIM
func (vu *volumeUtils) CreateVolume(bmcObj *infraiov1.Bmc, dispName string, namespacedName types.NamespacedName, updateVolumeObject bool, commonRec utils.ReconcilerInterface) (bool, error) {
	if updateVolumeObject {
		vu.commonRec.GetUpdatedVolumeObject(vu.ctx, namespacedName, vu.volObj)
	}
	//update track status
	volumeStatus := common.VolumeStatus{IsGettingCreated: true, IsGettingDeleted: false}
	vu.updateTrackVolume(bmcObj, volumeStatus)
	//create vol req body
	volReqBody := vu.GetVolumeRequestPayload(bmcObj.Status.BmcSystemID, dispName)
	var resp *http.Response
	var err error
	//send request for vol creation
	if vu.volObj.Spec.StorageControllerID == "" { // check added for reconcile volumes case
		resp, err = vu.volumeRestClient.Post(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes", bmcObj.Status.BmcSystemID, vu.volObj.Status.StorageControllerID), "Posting volume creation payload..", volReqBody)
	} else {
		resp, err = vu.volumeRestClient.Post(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes", bmcObj.Status.BmcSystemID, vu.volObj.Spec.StorageControllerID), "Posting volume creation payload..", volReqBody)
	}
	if err != nil {
		l.LogWithFields(vu.ctx).Error("error while creating volume:" + err.Error())
		return false, err
	}
	if resp.StatusCode == http.StatusAccepted {
		// BmcObj name is sent as ""
		done, _ := vu.commonUtil.MoniteringTaskmon(resp.Header, vu.ctx, common.CREATEVOLUME, vu.volObj.ObjectMeta.Name)
		if done {
			commonRec.UpdateBmcObjectOnReset(vu.ctx, bmcObj, fmt.Sprintf("%s %s Volume", constants.PendingForResetEvent, dispName))
			return true, nil
		}
	}
	l.LogWithFields(vu.ctx).Info(fmt.Sprintf("Could not create %s volume, try again", vu.volObj.ObjectMeta.Name))
	if updateVolumeObject {
		controllerutil.RemoveFinalizer(vu.volObj, VolFinalizer)
		err = vu.commonRec.GetCommonReconcilerClient().Update(vu.ctx, vu.volObj)
		if err != nil {
			l.LogWithFields(vu.ctx).Error("error while updating volume object:" + err.Error())
		}
		err = vu.commonRec.GetCommonReconcilerClient().Delete(vu.ctx, vu.volObj)
		if err != nil {
			l.LogWithFields(vu.ctx).Error("error while deleting volume object:" + err.Error())
		}
	}
	return false, err
}

// updateVolumeStatusAndClearSpec will clear the spec and upidate the status fields of volume object
func (vu *volumeUtils) UpdateVolumeStatusAndClearSpec(bmcObj *infraiov1.Bmc, dispName string) (bool, string) {
	getAllVolumesResp, sCode, err := vu.volumeRestClient.Get(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes", bmcObj.Status.BmcSystemID, vu.volObj.Spec.StorageControllerID), "Fetching all volumes..")
	volumeNotUpdatedMsg := "Could not update volume status, but volume is created"
	if err != nil {
		l.LogWithFields(vu.ctx).Error("error while fetching all volumes:" + err.Error())
		return false, volumeNotUpdatedMsg
	} else if sCode != http.StatusOK {
		l.LogWithFields(vu.ctx).Info("Failed to get all volumes, try again.")
		return false, volumeNotUpdatedMsg
	}
	volumesList := getAllVolumesResp["Members"].([]interface{})
	for _, vol := range volumesList {
		volURL := vol.(map[string]interface{})["@odata.id"].(string)
		getEachVolResp, sCode, err := vu.volumeRestClient.Get(volURL, fmt.Sprintf("Fetching %s volume details..", volURL[len(volURL)-1:]))
		if err != nil {
			l.LogWithFields(vu.ctx).Error(fmt.Sprintf("error while fetching %s volume details:", volURL[len(volURL)-1:]) + err.Error())
			return false, volumeNotUpdatedMsg
		} else if sCode != http.StatusOK {
			l.LogWithFields(vu.ctx).Info(fmt.Sprintf("Failed to get %s volume details, try again.", volURL[len(volURL)-1:]))
			return false, volumeNotUpdatedMsg
		} else {
			//getting all drives in the volume
			var found = false
			drives := []string{}
			driveLinks := getEachVolResp["Links"].(map[string]interface{})["Drives"].([]interface{})
			for _, dl := range driveLinks {
				drive := dl.(map[string]interface{})["@odata.id"].(string)
				driveURI := strings.Split(drive, "/")
				driveID := driveURI[len(driveURI)-1]
				if err != nil {
					l.Log.Info("Could not convert drive id to Integer")
				}
				drives = append(drives, driveID)
			}
			if getEachVolResp["RAIDType"] == vu.volObj.Spec.RAIDType && utils.CompareArray(drives, vu.volObj.Spec.Drives) {
				vu.volObj.ObjectMeta.Annotations["odata.id"] = getEachVolResp["@odata.id"].(string)
				err = vu.commonRec.GetCommonReconcilerClient().Update(vu.ctx, vu.volObj)
				if err != nil {
					l.LogWithFields(vu.ctx).Error(fmt.Sprintf("Error: Updating volume object %s annotations of bmc: %s", dispName, err.Error()))
				}
				identifiers := getEachVolResp["Identifiers"].([]interface{})
				var durableName, durableNameFormat string
				for _, id := range identifiers {
					durableName = id.(map[string]interface{})["DurableName"].(string)
					durableNameFormat = id.(map[string]interface{})["DurableNameFormat"].(string)
				}
				vu.commonRec.UpdateVolumeStatus(vu.ctx, vu.volObj, getEachVolResp["Id"].(string), getEachVolResp["Name"].(string), fmt.Sprintf("%f", getEachVolResp["CapacityBytes"].(float64)), durableName, durableNameFormat)
				vu.volObj.Spec = infraiov1.VolumeSpec{}
				err := vu.commonRec.GetCommonReconcilerClient().Update(vu.ctx, vu.volObj)
				if err != nil {
					l.LogWithFields(vu.ctx).Error(fmt.Sprintf("Error: Updating %s Volume object: %s", vu.volObj.Status.VolumeName, err.Error()))
				}
				found = true
				return true, "Successfully updated volume object status"
			}
			if !found {
				vu.commonRec.DeleteVolumeObject(vu.ctx, vu.volObj)
				return false, "Could not find the volume created in ODIM after reset, deleting the volume object, try creating again"
			}
		}
	}
	return false, volumeNotUpdatedMsg
}

// getVolumeRequestPayload prepares payload for volume creation
func (vu *volumeUtils) GetVolumeRequestPayload(systemID, dispName string) []byte {
	l.Log.Info("Creating volume request payload..")
	linkBody := link{}
	var reqBody requestPayload
	//check added to see if volume creation is triggered from reconcile, where volume object will already be present and we have to fetch details from status
	if vu.volObj.Spec.RAIDType == "" && vu.volObj.Spec.Drives == nil && vu.volObj.Spec.StorageControllerID == "" {
		for _, driveId := range vu.volObj.Status.Drives {
			// TODO : check if the drive is present and then proceed
			driveDet := map[string]string{"@odata.id": fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Drives/%s", systemID, vu.volObj.Status.StorageControllerID, driveId)}
			linkBody.Drives = append(linkBody.Drives, driveDet)
		}
		reqBody = requestPayload{RAIDType: vu.volObj.Status.RAIDType, Links: linkBody, DisplayName: dispName, ApplyTime: constants.ApplyTime}
	} else { // when volume is created via operator directly, it will pass through this case
		for _, driveId := range vu.volObj.Spec.Drives {
			// TODO : check if the drive is present and then proceed
			driveDet := map[string]string{"@odata.id": fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Drives/%s", systemID, vu.volObj.Spec.StorageControllerID, driveId)}
			linkBody.Drives = append(linkBody.Drives, driveDet)
		}
		reqBody = requestPayload{RAIDType: vu.volObj.Spec.RAIDType, Links: linkBody, DisplayName: dispName}
	}
	marshalBody, err := json.Marshal(reqBody)
	if err != nil {
		l.LogWithFields(vu.ctx).Error("Failed to marshal volume request payload" + err.Error())
	}
	return marshalBody
}

// DeleteVolume will delete volume in ODIM
func (vu *volumeUtils) DeleteVolume(bmcObj *infraiov1.Bmc) bool {
	//update volume status
	volumeStatus := common.VolumeStatus{IsGettingCreated: false, IsGettingDeleted: true}
	vu.updateTrackVolume(bmcObj, volumeStatus)
	deleteResp, err := vu.volumeRestClient.Delete(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/%s/Volumes/%s", bmcObj.Status.BmcSystemID, vu.volObj.Status.StorageControllerID, vu.volObj.Status.VolumeID), fmt.Sprintf("Deleting %s volume..", vu.volObj.Status.VolumeName))
	if err != nil {
		l.LogWithFields(vu.ctx).Error(fmt.Sprintf("error while deleting %s volume:", vu.volObj.Status.VolumeName) + err.Error())
		return false
	}
	if deleteResp.StatusCode == http.StatusAccepted {
		//monitering taskmon
		done, _ := vu.commonUtil.MoniteringTaskmon(deleteResp.Header, vu.ctx, common.DELETEVOLUME, vu.volObj.ObjectMeta.Name)
		if done {
			return true
		}
	}
	l.LogWithFields(vu.ctx).Info(fmt.Sprintf("Could not delete %s volume, try again", vu.volObj.ObjectMeta.Name))
	return false
}

// updateTrackVolume will update the TrackVolume map for polling
func (vu *volumeUtils) updateTrackVolume(bmcObj *infraiov1.Bmc, volumeStatus common.VolumeStatus) {
	var bmcStorageVolume common.BmcStorageControllerVolumesStatus
	if vu.volObj.Spec.StorageControllerID != "" { // While creation of volume,spec will be present,hence taking storageController from it,volume id is not present,hence using volObj name
		bmcStorageVolume = common.BmcStorageControllerVolumesStatus{Bmc: bmcObj.ObjectMeta.Name, StorageController: vu.volObj.Spec.StorageControllerID, VolumeID: vu.volObj.ObjectMeta.Name}
	} else { // when spec is empty, it means volume is already present,might be for deletion step,hence taking storageController and volume ID from status
		bmcStorageVolume = common.BmcStorageControllerVolumesStatus{Bmc: bmcObj.ObjectMeta.Name, StorageController: vu.volObj.Status.StorageControllerID, VolumeID: vu.volObj.Status.VolumeID}
	}
	common.TrackVolumeStatus[bmcStorageVolume] = volumeStatus
	l.LogWithFields(vu.ctx).Info("Track volume status: ", common.TrackVolumeStatus)
}

func (vu *volumeUtils) UpdateVolumeObject(volObject *infraiov1.Volume) {
	vu.volObj = volObject
}

// GetVolumeUtils will return volume utils object
func GetVolumeUtils(ctx context.Context, restClient restclient.RestClientInterface, commonRec utils.ReconcilerInterface, volObj *infraiov1.Volume, ns string) VolumeInterface {
	return &volumeUtils{
		ctx:              ctx,
		volumeRestClient: restClient,
		commonRec:        commonRec,
		volObj:           volObj,
		commonUtil:       common.GetCommonUtils(restClient),
		namespace:        ns,
	}
}
