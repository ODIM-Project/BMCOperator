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

// Package controllers ...
package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/google/uuid"
	Error "k8s.io/apimachinery/pkg/api/errors"
)

// FirmwareReconciler reconciles a Firmware object
type FirmwareReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var podName = os.Getenv("POD_NAME")

//+kubebuilder:rbac:groups=infra.io.odimra,resources=firmwares,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=firmwares/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=firmwares/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Firmware object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *FirmwareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//commonReconciler object creation
	transactionID := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionID.String(), constants.BmcOperator, constants.ODIMObjectOperationActionID, constants.ODIMObjectOperationActionName, podName)
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	firmObj := &infraiov1.Firmware{}
	// Fetch the Odim object
	err := r.Get(ctx, req.NamespacedName, firmObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if firmObj.ObjectMeta.Labels == nil || (firmObj.ObjectMeta.Annotations["old_firmware"] != "" && firmObj.ObjectMeta.Annotations["old_firmware"] != firmObj.Spec.Image.ImageLocation) {
		l.LogWithFields(ctx).Info("Firmware update starting...")
		odimObject := commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", req.Namespace)
		firmRestClient, err := restclient.NewRestClient(ctx, odimObject, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
		if err != nil {
			l.LogWithFields(ctx).Error("Failed to get rest client for ODIM" + err.Error())
			return ctrl.Result{}, err
		}
		//get firmwareUtil object
		firmUtil := GetFirmwareUtils(ctx, firmObj, commonRec, firmRestClient, req.Namespace)
		//build body
		firmwarePayload, err := firmUtil.getFirmwareBody()
		if err != nil {
			return ctrl.Result{}, nil
		}
		ok, isDelete, err := firmUtil.CreateFirmwareInSystem(firmwarePayload)
		if !ok {
			return ctrl.Result{}, nil
		} else if err != nil {
			return ctrl.Result{}, err
		} else if isDelete {
			l.LogWithFields(ctx).Info("Unable to update firmware, try again")
			err = r.Delete(ctx, firmObj)
			if err != nil {
				l.LogWithFields(ctx).Error("error while deleting firmware object" + err.Error())
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		time.Sleep(time.Duration(40) * time.Second) // added sleep so that fimware version is updated in /redfish/v1/Managers/
		firmVersion := firmUtil.GetFirmwareVersion()
		if firmVersion == "" {
			l.LogWithFields(ctx).Info("Not able to fetch updated firmware version from ODIM,hence skipping updating of firmware and bmc object")
			return ctrl.Result{}, nil
		}
		firmUtil.UpdateFirmwareLabels(firmVersion)
		commonRec.GetUpdatedFirmwareObject(ctx, req.NamespacedName, firmObj) // getting updated firmware object after updating labels to avoid updating on old firmware object
		firmUtil.UpdateFirmwareStatus("Success", firmVersion, firmObj.Spec.Image.ImageLocation)
		bmcObj := commonRec.GetBmcObject(ctx, constants.MetadataName, firmObj.Name, req.Namespace)
		time.Sleep(time.Duration(300) * time.Second)
		newSchema := firmUtil.checkIfBiosSchemaRegistryChanged(bmcObj)
		firmUtil.UpdateBmcObjectWithFirmwareVersionAndSchema(bmcObj, firmVersion, newSchema)
		l.LogWithFields(ctx).Info("Firmware and Bmc object updation completed!")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (fu *firmwareUtils) checkIfBiosSchemaRegistryChanged(bmcObj *infraiov1.Bmc) string {
	allBiosSchemaRegistryObjects := fu.commonRec.GetAllBiosSchemaRegistryObjects(fu.ctx, fu.namespace)
	allBiosSchemaRegistryObjectsNames := getBiosSchemaRegistryObjectNames(allBiosSchemaRegistryObjects)
	var biosAttributeID string
	var registryResponse map[string]interface{}
	systemDetails := fu.commonUtil.GetBmcSystemDetails(fu.ctx, bmcObj)
	if biosVersion, ok := systemDetails["BiosVersion"].(string); ok {
		biosAttributeID, registryResponse = fu.biosUtil.GetBiosAttributeID(biosVersion, bmcObj)
	}
	if biosAttributeID != "" {
		formattedBiosAttributeID := utils.RemoveSpecialChar(biosAttributeID)
		for _, schemaName := range allBiosSchemaRegistryObjectsNames {
			if formattedBiosAttributeID != schemaName {
				l.LogWithFields(fu.ctx).Info("Bios Schema Registry has changed, pulling the new schema...")
				created := fu.commonRec.CheckAndCreateBiosSchemaObject(fu.ctx, registryResponse, bmcObj)
				if created {
					l.LogWithFields(fu.ctx).Info(fmt.Sprintf("New Bios Schema Registry object %s created", schemaName))
					return biosAttributeID
				}
			} else {
				l.LogWithFields(fu.ctx).Info("Bios Schema Registry has not changed")
			}
		}
	} else {
		l.LogWithFields(fu.ctx).Info("Not able to find the Bios Version")
	}
	return ""
}

func getBiosSchemaRegistryObjectNames(allBiosSchemaRegistryObjects *[]infraiov1.BiosSchemaRegistry) []string {
	allBiosRegistryFormattedNames := []string{}
	for _, biosRegistry := range *allBiosSchemaRegistryObjects {
		allBiosRegistryFormattedNames = append(allBiosRegistryFormattedNames, biosRegistry.ObjectMeta.Name)
	}
	return allBiosRegistryFormattedNames
}

func (fu *firmwareUtils) CreateFirmwareInSystem(firmwarePayload []byte) (bool, bool, error) {
	firmwareResp, err := fu.firmRestClient.Post("/redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate", "Posting firmware update..", firmwarePayload)
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error while firmware update" + err.Error())
		return false, false, err
	}
	if firmwareResp.StatusCode == http.StatusAccepted {
		done, _ := fu.commonUtil.MoniteringTaskmon(fu.ctx, firmwareResp.Header, common.FIRMWARE, fu.firmObj.ObjectMeta.Name)
		if done {
			time.Sleep(time.Duration(40) * time.Second)
			bmcObj := fu.commonRec.GetBmcObject(fu.ctx, constants.MetadataName, fu.firmObj.Name, fu.firmObj.Namespace)
			if ok := common.MapOfFirmware[bmcObj.GetName()]; ok {
				delete(common.MapOfFirmware, bmcObj.GetName())
			}
			return true, false, nil
		}
	}
	return false, true, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FirmwareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.Firmware{}).
		WithEventFilter(utils.IgnoreStatusUpdate()).
		Complete(r)
}

// UpdateBmcObjectWithFirmwareVersion updates the bmc object with latest firmware version
func (fu *firmwareUtils) UpdateBmcObjectWithFirmwareVersionAndSchema(bmcObj *infraiov1.Bmc, firmVersion, biosSchema string) {
	bmcObj.Status.FirmwareVersion = firmVersion
	if biosSchema != "" {
		bmcObj.Status.BiosAttributeRegistry = biosSchema
	}
	err := fu.commonRec.GetCommonReconcilerClient().Status().Update(fu.ctx, bmcObj)
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error: Updating firmware in bmc status " + err.Error())
	}
	time.Sleep(time.Duration(40) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
	firmwareVersion := formatForLabels(firmVersion)
	bmcObj.ObjectMeta.Labels["firmwareVersion"] = firmwareVersion
	err = fu.commonRec.GetCommonReconcilerClient().Update(fu.ctx, bmcObj)
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error: Updating firmware bmc labels " + err.Error())
	}
	time.Sleep(time.Duration(40) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
}

// UpdateFirmwareLabels updates the labels of firmware object
func (fu *firmwareUtils) UpdateFirmwareLabels(firmVersion string) {
	fu.firmObj.ObjectMeta.Labels = map[string]string{}
	firmwareVersion := formatForLabels(firmVersion)
	fu.firmObj.ObjectMeta.Labels["firmwareVersion"] = firmwareVersion
	fu.firmObj.ObjectMeta.Annotations["old_firmware"] = fu.firmObj.Spec.Image.ImageLocation
	err := fu.commonRec.GetCommonReconcilerClient().Update(fu.ctx, fu.firmObj)
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error: Updating firmware operation " + err.Error())
	}
	time.Sleep(time.Duration(40) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
}

// formatForLabels will remove all the spaces and replace it with '-' since labels do not allow spaces
func formatForLabels(firmVersion string) string {
	firmStrArr := strings.Split(firmVersion, " ")
	firmwareVersion := strings.Join(firmStrArr, "-")
	return firmwareVersion
}

// UpdateFirmwareStatus updates the firmware status
func (fu *firmwareUtils) UpdateFirmwareStatus(status, firmVersion, imageLocation string) {
	fu.firmObj.Status.Status = status
	if firmVersion != "" {
		fu.firmObj.Status.FirmwareVersion = firmVersion
		fu.firmObj.Status.ImagePath = imageLocation
	}
	err := fu.commonRec.GetCommonReconcilerClient().Status().Update(fu.ctx, fu.firmObj)
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error: Updating firmware operation " + err.Error())
	}
	time.Sleep(time.Duration(40) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
}

func (fu *firmwareUtils) GetFirmwareVersion() string {
	systemID := fu.GetSystemID()
	managerResp, _, err := fu.firmRestClient.Get(fmt.Sprintf("/redfish/v1/Managers/%s", systemID), "Getting firmware version details..")
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error: Fetching firmware version details " + err.Error())
		return ""
	}
	inventories := fu.GetFirmwareInventoryDetails(systemID)
	fu.firmObj.ObjectMeta.Annotations["odata.id"] = strings.Join(inventories, ",")
	err = fu.commonRec.GetCommonReconcilerClient().Update(fu.ctx, fu.firmObj)
	if err != nil {
		l.LogWithFields(fu.ctx).Error(fmt.Sprintf("Error: Updating firmware object annotations: %s of bmc", err.Error()))
	}
	if _, ok := managerResp["FirmwareVersion"].(string); ok {
		return managerResp["FirmwareVersion"].(string)
	}
	return ""
}

// getFirmwareBody prepares the payload for firmware operation
func (fu *firmwareUtils) getFirmwareBody() ([]byte, error) {
	var body interface{}
	var systemsURI string
	if systemID := fu.GetSystemID(); systemID != "" {
		systemsURI = fmt.Sprintf("/redfish/v1/Systems/%s", systemID)
	} else {
		l.LogWithFields(fu.ctx).Info("BMC object doesn't exist for firmware update, Please check if it is added.")
		fu.commonRec.GetCommonReconcilerClient().Delete(fu.ctx, fu.firmObj)
		return nil, errors.New("bmc object doesnt exist")
	}
	if fu.firmObj.Spec.Image.Auth.Username == "" || fu.firmObj.Spec.Image.Auth.Password == "" && fu.firmObj.Spec.TransferProtocol == "" {
		body = FirmPayloadWithoutAuth{Image: fu.firmObj.Spec.Image.ImageLocation, Targets: []string{systemsURI}}
	} else if fu.firmObj.Spec.TransferProtocol == "" {
		body = firmPayload{Image: fu.firmObj.Spec.Image.ImageLocation, Targets: []string{systemsURI}, Username: fu.firmObj.Spec.Image.Auth.Username, Password: fu.firmObj.Spec.Image.Auth.Password}
	} else {
		body = firmPayloadWithTransferProtocol{Image: fu.firmObj.Spec.Image.ImageLocation, Targets: []string{systemsURI}, Username: fu.firmObj.Spec.Image.Auth.Username, Password: fu.firmObj.Spec.Image.Auth.Password, TransferProtocol: fu.firmObj.Spec.TransferProtocol}
	}
	marshalInput, err := json.Marshal(body)
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Failed to marshal firmware input" + err.Error())
		return nil, err
	}
	return marshalInput, nil
}

// GetSystemID will get the systemID of the bmc whose firmware needs to be updated
func (fu *firmwareUtils) GetSystemID() string {
	bmcObj := fu.commonRec.GetBmcObject(fu.ctx, constants.MetadataName, fu.firmObj.Name, fu.namespace)
	if bmcObj != nil {
		return bmcObj.Status.BmcSystemID
	}
	return ""
}

func (fu *firmwareUtils) GetFirmwareInventoryDetails(systemID string) []string {
	systemUUID := strings.Split(systemID, ".")
	inventoryList := []string{}
	firmwareInventoryResp, _, err := fu.firmRestClient.Get("/redfish/v1/UpdateService/FirmwareInventory", "Getting firmware inventory details..")
	if err != nil {
		l.LogWithFields(fu.ctx).Error("Error: Fetching firmware inventory details " + err.Error())
		return []string{}
	}

	if inventories, ok := firmwareInventoryResp["Members"].([]interface{}); ok {
		for _, member := range inventories {
			odataid := member.(map[string]interface{})["@odata.id"].(string)
			if strings.Contains(odataid, systemUUID[0]) {
				inventoryList = append(inventoryList, odataid)
			}
		}
	}
	return inventoryList
}

// GetFirmwareUtils returns firmwareUtil
func GetFirmwareUtils(ctx context.Context, firmObj *infraiov1.Firmware, commonRec utils.ReconcilerInterface, firmRestClient restclient.RestClientInterface, ns string) firmwareInterface {
	return &firmwareUtils{
		ctx:            ctx,
		firmObj:        firmObj,
		commonRec:      commonRec,
		biosUtil:       bios.GetBiosUtils(ctx, nil, commonRec, firmRestClient, ns),
		firmRestClient: firmRestClient,
		commonUtil:     common.GetCommonUtils(firmRestClient),
		namespace:      ns,
	}
}
