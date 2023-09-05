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
)

// BootOrderSettingsReconciler reconciles a BootOrderSetting object
type BootOrderSettingsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var podName = os.Getenv("POD_NAME")

//+kubebuilder:rbac:groups=infra.io.odimra,resources=bootordersettings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=bootordersettings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=bootordersettings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BootOrderSetting object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *BootOrderSettingsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	transactionId := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionId.String(), constants.BmcOperator, constants.BootOrderActionID, constants.BootOrderActionName, podName)
	bootObj := &infraiov1.BootOrderSetting{}
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	err := r.Get(ctx, req.NamespacedName, bootObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	odimObject := commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", req.Namespace)
	bootRestClient, err := restclient.NewRestClient(ctx, odimObject, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get rest client for BMC: %s", err.Error())
		return ctrl.Result{}, err
	}
	bootUtil := GetBootUtils(ctx, bootObj, commonRec, bootRestClient, common.GetCommonUtils(bootRestClient), req.Namespace)
	if bootObj.Spec.Boot != nil && (len(bootObj.Spec.Boot.BootTargetAllowableValues) > 0 || len(bootObj.Spec.Boot.UefiTargetAllowableValues) > 0 ||
		bootObj.Spec.Boot.BootSourceOverrideMode != "") {
		l.LogWithFields(ctx).Info("Unmodify values passed in input")

		return ctrl.Result{}, nil
	}
	if bootObj.Spec.BmcName != "" || bootObj.Spec.SerialNo != "" || bootObj.Spec.SystemID != "" {
		if bootObj.Spec.Boot != nil && (len(bootObj.Spec.Boot.BootOrder) > 0 || bootObj.Spec.Boot.BootSourceOverrideEnabled != "" ||
			bootObj.Spec.Boot.BootSourceOverrideTarget != "" || bootObj.Spec.Boot.UefiTargetBootSourceOverride != "") {
			res := bootUtil.updateBootSettings()
			if !res {
				return ctrl.Result{}, nil
			}
		}
	}

	return ctrl.Result{}, nil
}

// updateBootSettings is used to update the boot order setting with input given by user
func (bo *bootUtils) updateBootSettings() bool {
	l.LogWithFields(bo.ctx).Info("Set boot order settings for BMC")
	var systemID, bootBmcIP string
	var bootBmcObj *infraiov1.Bmc
	if bo.bootObj.Spec.BmcName != "" {
		bootBmcObj = bo.commonRec.GetBmcObject(bo.ctx, "spec.bmc.address", bo.bootObj.Spec.BmcName, bo.bootObj.Namespace)
		if bootBmcObj.Spec.BmcDetails.Address == "" {
			l.LogWithFields(bo.ctx).Info("Check if BMC is registered..")
			return false
		}
		systemID = bootBmcObj.Status.BmcSystemID
	} else if bo.bootObj.Spec.SystemID != "" {
		bootBmcObj = bo.commonRec.GetBmcObject(bo.ctx, constants.StatusBmcSystemID, bo.bootObj.Spec.SystemID, bo.bootObj.Namespace)
		if bootBmcObj.Status.BmcSystemID == "" {
			l.LogWithFields(bo.ctx).Info("Check if BMC is registered..")
			return false
		}
		systemID = bo.bootObj.Spec.SystemID
	} else if bo.bootObj.Spec.SerialNo != "" {
		bootBmcObj = bo.commonRec.GetBmcObject(bo.ctx, "status.serialNumber", bo.bootObj.Spec.SerialNo, bo.bootObj.Namespace)
		if bootBmcObj.Status.SerialNumber == "" {
			l.LogWithFields(bo.ctx).Info("Check if BMC is registered..")
			return false
		}
	}
	bootBmcIP = bootBmcObj.Spec.BmcDetails.Address
	systemID = bootBmcObj.Status.BmcSystemID
	ok := bo.UpdateBootDetails(bo.ctx, systemID, bootBmcIP, bo.bootObj, bootBmcObj)
	if ok {
		sysDetails := bo.commonUtil.GetBmcSystemDetails(bo.ctx, bootBmcObj)
		bo.bootObj.ObjectMeta.Annotations["odata.id"] = getBootOID(sysDetails)
		err := bo.commonRec.GetCommonReconcilerClient().Update(bo.ctx, bo.bootObj)
		if err != nil {
			l.LogWithFields(bo.ctx).Error(fmt.Sprintf("Error: Updating %s boot order setting object annotations: %s of bmc", bootBmcObj.Name, err.Error()))
		}
		bootSetting := bo.GetBootAttributes(sysDetails)
		bo.bootObj.Spec = infraiov1.BootOrderSettingsSpec{}
		bo.commonRec.GetCommonReconcilerClient().Update(bo.ctx, bo.bootObj)
		bo.bootObj.Status.Boot = *bootSetting
		err = bo.commonRec.GetCommonReconcilerClient().Status().Update(bo.ctx, bo.bootObj)
		if err != nil {
			l.LogWithFields(bo.ctx).Error(fmt.Sprintf("Error while updating boot order setting object for %s BMC: %s", bootBmcObj.Name, err.Error()))
			return false
		}
		l.LogWithFields(bo.ctx).Info(fmt.Sprintf("Boot order setting configured for %s BMC", bootBmcObj.Name))
		return true
	}
	return false
}

func getBootOID(sysDetails map[string]interface{}) string {
	boot := sysDetails["Boot"].(map[string]interface{})
	bootOptions := boot["BootOptions"].(map[string]interface{})
	return bootOptions["@odata.id"].(string)
}

// UpdateBootDetails function is used to update the boot details in server
func (bo *bootUtils) UpdateBootDetails(ctx context.Context, systemID, bootBmcIP string, bootObj *infraiov1.BootOrderSetting, bootBmcObj *infraiov1.Bmc) bool {
	var boot = Boot{}
	if len(bootObj.Spec.Boot.BootOrder) > 0 {
		if !utils.CompareArray(bootObj.Status.Boot.BootOrder, bootObj.Spec.Boot.BootOrder) {
			l.LogWithFields(bo.ctx).Error("Invalid value passed for BootOrder")
			return false
		}
		boot.BootOrder = bootObj.Spec.Boot.BootOrder
	}
	if bootObj.Spec.Boot.BootSourceOverrideTarget != "" {
		if !utils.ContainsValue(bootObj.Status.Boot.BootTargetAllowableValues, bootObj.Spec.Boot.BootSourceOverrideTarget) {
			l.LogWithFields(bo.ctx).Error(fmt.Sprintf("Invalid value passed for BootSourceOverrideTarget for %s BMC", bootBmcIP))
			return false
		}
		boot.BootSourceOverrideTarget = bootObj.Spec.Boot.BootSourceOverrideTarget
	}
	if bootObj.Spec.Boot.BootSourceOverrideEnabled != "" {
		boot.BootSourceOverrideEnabled = bootObj.Spec.Boot.BootSourceOverrideEnabled
	}
	if bootObj.Spec.Boot.UefiTargetBootSourceOverride != "" {
		if !utils.ContainsValue(bootObj.Status.Boot.UefiTargetAllowableValues, bootObj.Spec.Boot.UefiTargetBootSourceOverride) {
			l.LogWithFields(bo.ctx).Error(fmt.Sprintf("Invalid value passed for UefiTargetBootSourceOverride for %s BMC", bootBmcIP))
			return false
		}
		boot.UefiTargetBootSourceOverride = bootObj.Spec.Boot.UefiTargetBootSourceOverride
	}
	bootSetting := BootOrderSetting{
		Boot: boot,
	}
	bootBody, err := json.Marshal(bootSetting)
	if err != nil {
		l.LogWithFields(bo.ctx).Error(fmt.Sprintf("Error marshalling boot request body for %s BMC: %s", bootBmcObj.Name, err.Error()))
		bootBody = nil
	}
	bootLink := "/redfish/v1/Systems/" + systemID
	patchResponse, err := bo.bootRestClient.Patch(bootLink, "Patching boot payload..", bootBody)
	if err == nil && patchResponse.StatusCode == http.StatusAccepted {
		ok, _ := bo.commonUtil.MoniteringTaskmon(patchResponse.Header, ctx, common.BOOTSETTING, bootBmcObj.Name)
		return ok
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *BootOrderSettingsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.BootOrderSetting{}).
		Complete(r)
}

// updateBootAttributesOnReset updates the boot attributes with latest values
func (bo *bootUtils) UpdateBootAttributesOnReset(bmcName string, bootSetting *infraiov1.BootSetting) {
	bootObj := bo.commonRec.GetBootObject(bo.ctx, constants.MetadataName, bmcName, bo.namespace)
	if bootObj != nil {
		bootObj.Status.Boot = *bootSetting
	}
	err := bo.commonRec.GetCommonReconcilerClient().Status().Update(bo.ctx, bootObj)
	if err != nil {
		l.LogWithFields(bo.ctx).Error(fmt.Sprintf("Error: while updating boot order object for %s BMC: %s", bootObj.ObjectMeta.Name, err.Error()))
	}
}

// getBootAttributes is used to get boot attributes value from bmc
func (bo *bootUtils) GetBootAttributes(sysDetails map[string]interface{}) *infraiov1.BootSetting {
	var bootOrder, targetAllowableVal, uefiAllowableVal []string
	bootDetails := infraiov1.BootSetting{}
	if sysDetails != nil {
		boot := sysDetails["Boot"].(map[string]interface{})
		if boot["BootOrder"].([]interface{}) != nil {
			bootOrder = utils.ConvertToArray(boot["BootOrder"].([]interface{}))
		}
		if boot["BootSourceOverrideTarget@Redfish.AllowableValues"] != nil {
			targetAllowableVal = utils.ConvertToArray(boot["BootSourceOverrideTarget@Redfish.AllowableValues"].([]interface{}))
		}
		if boot["UefiTargetBootSourceOverride@Redfish.AllowableValues"] != nil {
			uefiAllowableVal = utils.ConvertToArray(boot["UefiTargetBootSourceOverride@Redfish.AllowableValues"].([]interface{}))
		}
		bootDetails = infraiov1.BootSetting{
			BootOrder:                 bootOrder,
			BootTargetAllowableValues: targetAllowableVal,
			UefiTargetAllowableValues: uefiAllowableVal,
		}
		if val, ok := boot["BootSourceOverrideEnabled"]; ok {
			if s, ok := val.(string); ok {
				bootDetails.BootSourceOverrideEnabled = s
			}
		}
		if val, ok := boot["BootSourceOverrideMode"]; ok {
			if s, ok := val.(string); ok {
				bootDetails.BootSourceOverrideMode = s
			}
		}
		if val, ok := boot["UefiTargetBootSourceOverride"]; ok {
			if s, ok := val.(string); ok {
				bootDetails.UefiTargetBootSourceOverride = s
			}
		}
		if val, ok := boot["BootSourceOverrideTarget"]; ok {
			if s, ok := val.(string); ok {
				bootDetails.BootSourceOverrideTarget = s
			}
		}
	}
	return &bootDetails
}

func GetBootUtils(ctx context.Context, bootObj *infraiov1.BootOrderSetting, commonRec utils.ReconcilerInterface, bootRestClient restclient.RestClientInterface, commonUtil common.CommonInterface, ns string) BootInterface {
	return &bootUtils{
		ctx:            ctx,
		bootObj:        bootObj,
		commonRec:      commonRec,
		bootRestClient: bootRestClient,
		commonUtil:     commonUtil,
		namespace:      ns,
	}
}
