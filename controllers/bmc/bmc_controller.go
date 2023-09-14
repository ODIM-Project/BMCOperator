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
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	Error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	boot "github.com/ODIM-Project/BMCOperator/controllers/boot"
	"github.com/google/uuid"

	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	eventsubscription "github.com/ODIM-Project/BMCOperator/controllers/eventsubscription"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	volume "github.com/ODIM-Project/BMCOperator/controllers/volume"
	l "github.com/ODIM-Project/BMCOperator/logs"
)

// BmcReconciler reconciles a Bmc object
type BmcReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var podName = os.Getenv("POD_NAME")

//+kubebuilder:rbac:groups=infra.io.odimra,resources=bmcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=bmcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=bmcs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Bmc object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *BmcReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	transactionId := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionId.String(), constants.BmcOperator, constants.BMCSettingActionID, constants.BMCSettingActionName, podName)
	var updateBMCObject bool
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	odimObject := commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", req.Namespace) //field name has to be taken from odim object
	bmcRestClient, err := restclient.NewRestClient(ctx, odimObject, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get rest client for BMC: %s", err.Error())
		return ctrl.Result{}, err
	}
	// Fetch the Bmc instance.
	bmcObj := &infraiov1.Bmc{}
	err = r.Get(ctx, req.NamespacedName, bmcObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	commonUtil := common.GetCommonUtils(bmcRestClient)
	bmcUtil := GetBmcUtils(ctx, bmcObj, req.Namespace, &commonRec, &bmcRestClient, commonUtil, updateBMCObject)
	//get config details

	if err != nil {
		l.LogWithFields(ctx).Errorf("Unable to get configuration details, please check if config.yaml is applied: %s", err.Error())
		return ctrl.Result{}, err
	}
	//get keys from secret
	secret := &corev1.Secret{}
	encryptedPrivateKey, encryptedPublicKey := GetEncryptedPemKeysFromSecret(ctx, secret, config.Data.SecretName, config.Data.Namespace, r.Client)
	privateKey, publicKey := GetPemKeys(ctx, encryptedPrivateKey, encryptedPublicKey)
	if privateKey == nil || publicKey == nil {
		l.LogWithFields(ctx).Info("Could not find the private and public keys, please apply the secret and try again")
		commonRec.DeleteBmcObject(ctx, bmcObj)
		return ctrl.Result{}, nil
	}
	//Checking for deletion
	isBmcMarkedToBeDeleted := bmcObj.GetDeletionTimestamp() != nil
	if isBmcMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(bmcObj, bmcFinalizer) {
			isBmcDeleted := bmcUtil.unregisterBmc()
			if isBmcDeleted {
				bmcUtil.removeBmcFinalizerAndDeleteDependencies()
				err := r.Update(ctx, bmcObj)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else {
				l.LogWithFields(ctx).Info(fmt.Sprintf("BMC %s not deleted! Retrying..", bmcObj.Spec.BmcDetails.Address))
				return ctrl.Result{Requeue: true}, nil // Calling Reconcile again for delete to execute
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(bmcObj, bmcFinalizer) {
		controllerutil.AddFinalizer(bmcObj, bmcFinalizer)
		updateBMCObject = true
	}

	//Len check: to check if user modified the encrypted pass for password change reconcile to run
	// SystemResetStatus check : To stop reconcile flow through add bmc code for bios reset change
	if len(bmcObj.Spec.Credentials.Password) < 20 && bmcObj.Status.SystemReset != fmt.Sprintf("%s Bios", constants.PendingForResetEvent) {
		updateBMCObject = true
		if bmcObj.ObjectMeta.Annotations["old_password"] != "" {
			var updatedInBmc, updatedInOdim bool
			//decrypt the encrypted old pass
			decryptedPass := utils.DecryptWithPrivateKey(ctx, bmcObj.ObjectMeta.Annotations["old_password"], *utils.PrivateKey, true) //doBase64Decode = true, because bmc password is encrypted n encoded
			// update password change
			if decryptedPass != bmcObj.Spec.Credentials.Password {
				//update bmc with new pass
				updatedInBmc, err = bmcUtil.updateBmcWithNewPassword()
				if err != nil || !updatedInBmc {
					encryptedPassword := utils.EncryptWithPublicKey(ctx, decryptedPass, *utils.PublicKey)
					if encryptedPassword != "" {
						bmcObj.Spec.Credentials.Password = encryptedPassword
					}
				} else {
					//update odim with new pass if its updated in bmc
					updatedInOdim, err = bmcUtil.updateOdimWithNewPassword()
					if err != nil || !updatedInOdim {
						//check this case
						if err != nil {
							l.LogWithFields(ctx).Errorf("Error: Updating new password in Odim for %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error())
						} else {
							l.LogWithFields(ctx).Info(fmt.Sprintf("Updating new password in Odim for %s BMC failed, Try again", bmcObj.Spec.BmcDetails.Address))
						}
						encryptedPassword := utils.EncryptWithPublicKey(ctx, decryptedPass, *utils.PublicKey)
						if encryptedPassword != "" {
							bmcObj.Spec.Credentials.Password = encryptedPassword
						}
					} else {
						l.LogWithFields(ctx).Info(fmt.Sprintf("Updated password in ODIM for %s BMC", bmcObj.Spec.BmcDetails.Address))
					}
					//update annotations with old password and encrypt exisiting password
					if updatedInBmc && updatedInOdim {
						l.LogWithFields(ctx).Info(fmt.Sprintf("Updating old password and encrypting current password for %s BMC", bmcObj.Spec.BmcDetails.Address))
						bmcUtil.encryptPassword(utils.PublicKey)
						l.LogWithFields(ctx).Info(fmt.Sprintf("Successfully completed updating password for %s BMC!", bmcObj.Spec.BmcDetails.Address))
					}
				}
			} else {
				l.LogWithFields(ctx).Info(fmt.Sprintf("Password set is the same as before for %s BMC", bmcObj.Spec.BmcDetails.Address))
				bmcUtil.encryptPassword(utils.PublicKey)
			}
		} else {
			// retrive conn method for bmc
			// odimObj := commonRec.GetOdimObject(constants.MetadataName, "odim", req.Namespace)
			connMeth := bmcUtil.GetConnectionMethod(odimObject)
			if connMeth != "" {
				body, err := bmcUtil.PrepareAddBmcPayload(connMeth)
				if err != nil {
					l.LogWithFields(ctx).Errorf("Error: Creating the request body for adding %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error())
				}
				// add bmc
				added := bmcUtil.addBmc(body, req.NamespacedName, bmcObj.Labels["systemId"])
				//update annotations with old password and encrypt exisiting password if bmc added
				if added {
					// check if Annotations is not equal to nil in case of reconciliation
					if bmcObj.ObjectMeta.Annotations != nil {
						bmcObj.ObjectMeta.Annotations["old_password"] = ""
						l.LogWithFields(ctx).Info(fmt.Sprintf("Updating old password and encrypting Current password for %s BMC", bmcObj.Spec.BmcDetails.Address))
						bmcUtil.encryptPassword(publicKey)
					}
					// update labels
					l.LogWithFields(ctx).Info("Updating labels..")
					bmcUtil.updateBmcLabels()
					l.LogWithFields(ctx).Info("Successfully completed")
				} else {
					l.LogWithFields(ctx).Info(fmt.Sprintf("BMC %s not added, try again", bmcObj.Spec.BmcDetails.Address))
					bmcUtil.deleteBMCObject()
					return ctrl.Result{}, nil
				}
			} else {
				l.LogWithFields(ctx).Info(fmt.Sprintf("Odim doesn't support the plugin required for this %s Bmc!", bmcObj.Spec.BmcDetails.Address))
				updateBMCObject = false
				bmcUtil.deleteBMCObject()
			}
		}
	}
	// reset code
	if bmcObj.Spec.BmcDetails.PowerState != "" && bmcObj.Spec.BmcDetails.ResetType != "" {
		updateBMCObject = true
		var currentPowerState string
		allowableVal := bmcUtil.getAllowableResetValues()
		if allowableVal == nil {
			l.LogWithFields(ctx).Info(fmt.Sprintf("Allowable values not available for %s BMC", bmcObj.Spec.BmcDetails.Address))
			return ctrl.Result{}, nil
		}
		sysRes := bmcUtil.(*bmcUtils).commonUtil.GetBmcSystemDetails(ctx, bmcObj) //GetSystemDetails()
		if val, ok := sysRes["PowerState"]; ok {
			currentPowerState = val.(string)
			l.LogWithFields(ctx).Info(fmt.Sprintf("Current power state for %s BMC: `%s`", bmcObj.Spec.BmcDetails.Address, currentPowerState))
		}
		isValid := bmcUtil.validateResetData(currentPowerState, allowableVal)
		if !isValid {
			bmcObj.Spec.BmcDetails.PowerState = ""
			bmcObj.Spec.BmcDetails.ResetType = ""
		} else {
			resetDone := bmcUtil.ResetSystem(false, true)
			if resetDone {
				bmcObj.Status.SystemReset = "Done"
				bmcObj.Status.PowerState = bmcObj.Spec.BmcDetails.PowerState
				commonRec.UpdateBmcStatus(ctx, bmcObj)
				l.LogWithFields(ctx).Info(fmt.Sprintf("successfully done computer system reset on %s BMC", bmcObj.Spec.BmcDetails.Address))
			}
			bmcObj.Spec.BmcDetails.PowerState = ""
			bmcObj.Spec.BmcDetails.ResetType = ""
		}
	}
	annotations := bmcObj.GetAnnotations()
	if _, exists := annotations["kubectl.kubernetes.io/last-applied-configuration"]; exists {
		// Remove the last-applied-configuration annotation if it exists.
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		bmcObj.SetAnnotations(annotations)
		updateBMCObject = true
	}
	//flag to update bmc object after all modifications
	if updateBMCObject {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Updating %s BMC object...", bmcObj.Spec.BmcDetails.Address))
		err := r.Client.Update(ctx, bmcObj)
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating %s BMC object: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
			return ctrl.Result{}, err
		}
		time.Sleep(time.Duration(constants.SleepTime) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
		l.LogWithFields(ctx).Info(fmt.Sprintf("%s BMC object is updated.", bmcObj.Spec.BmcDetails.Address))
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BmcReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.Bmc{}).
		WithEventFilter(utils.IgnoreStatusUpdate()).
		Complete(r)
}

// --------Add BMC------
// addBmc used to add BMC
func (bu *bmcUtils) addBmc(body []byte, namespaceName types.NamespacedName, sysID string) bool {
	biosUtil := bios.GetBiosUtils(bu.ctx, nil, bu.commonRec, bu.bmcRestClient, bu.namespace) //getting biosUtil
	bootUtil := boot.GetBootUtils(bu.ctx, nil, bu.commonRec, bu.bmcRestClient, bu.commonUtil, bu.namespace)
	eventSubUtils := eventsubscription.GetEventSubscriptionUtils(bu.ctx, bu.bmcRestClient, bu.commonRec, nil, bu.namespace)
	l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Adding %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
	if sysID == "" {
		done, taskResp := bu.commonUtil.BmcAddition(bu.ctx, bu.bmcObj, body, bu.bmcRestClient)
		if done {
			if taskResp != nil && taskResp["Name"].(string) == "Aggregation Source" {
				return updateBmcDetails(taskResp["Id"].(string), bu, biosUtil, bootUtil, eventSubUtils)
			}
		}
	} else {
		return updateBmcDetails(sysID, bu, biosUtil, bootUtil, eventSubUtils)
	}
	return false
}

// updateBmcDetails is used to get bmc details and update in kubernetes object
func updateBmcDetails(sysId string, bu *bmcUtils, biosUtil bios.BiosInterface, bootUtil boot.BootInterface, eventSubUtils eventsubscription.EventSubscriptionInterface) bool {
	bu.bmcObj.Status.BmcAddStatus = "yes"
	bu.bmcObj.Status.BmcSystemID = sysId
	l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Systems ID for %s Bmc is %s", bu.bmcObj.Spec.BmcDetails.Address, sysId))
	systemDetails := bu.commonUtil.GetBmcSystemDetails(bu.ctx, bu.bmcObj)
	mgrDetails := bu.GetManagerDetails()
	if systemDetails != nil {
		bu.bmcObj.ObjectMeta.Annotations["odata.id"] = systemDetails["@odata.id"].(string)
		err := bu.commonRec.GetCommonReconcilerClient().Update(bu.ctx, bu.bmcObj)
		if err != nil {
			l.LogWithFields(bu.ctx).Error(fmt.Sprintf("Error: Updating %s BMC object : %s", bu.bmcObj.Spec.BmcDetails.Address, err.Error()))
		}

		bu.bmcObj.Status.BmcAddStatus = "yes"
		bu.bmcObj.Status.BmcSystemID = sysId
		if serialNumber, ok := systemDetails["SerialNumber"].(string); ok {
			bu.bmcObj.Status.SerialNumber = serialNumber
		}
		if powerState, ok := systemDetails["PowerState"].(string); ok {
			bu.bmcObj.Status.PowerState = powerState
		}
		if vendorName, ok := systemDetails["Manufacturer"].(string); ok {
			val, vOk := utils.CaseInsensitiveContains(vendorName)
			if vOk {
				bu.bmcObj.Status.VendorName = val
			}
		}
		if modelID, ok := systemDetails["Model"].(string); ok {
			bu.bmcObj.Status.ModelID = modelID
		}
		if biosVersion, ok := systemDetails["BiosVersion"].(string); ok {
			bu.bmcObj.Status.BiosVersion = biosVersion
			biosID, registryResp := biosUtil.GetBiosAttributeID(biosVersion, bu.bmcObj)
			if biosID != "" {
				bu.bmcObj.Status.BiosAttributeRegistry = biosID
				bu.commonRec.CheckAndCreateBiosSchemaObject(bu.ctx, registryResp, bu.bmcObj)
			}
		}
		bootAttribute := bootUtil.GetBootAttributes(systemDetails)
		biosAttribute := biosUtil.GetBiosAttributes(bu.bmcObj)
		isObjCreated := bu.commonRec.CreateBiosSettingObject(bu.ctx, biosAttribute, bu.bmcObj)
		if isObjCreated {
			l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Bios setting object created for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
		}
		if bootAttribute != nil {
			isBootObjCreated := bu.commonRec.CreateBootOrderSettingObject(bu.ctx, bootAttribute, bu.bmcObj)
			if isBootObjCreated {
				l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Boot order object created for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
			}
		}
	}

	var registryID []string
	var messageRegistryResp []map[string]interface{}
	if systemDetails["Manufacturer"].(string) == constants.ILOManufacturer {
		registryID, messageRegistryResp = eventSubUtils.GetMessageRegistryDetails(bu.bmcObj, "iLOEvents")
	}

	if len(registryID) > 0 {
		bu.bmcObj.Status.EventsMessageRegistry = strings.Join(registryID, ",")
		for _, resp := range messageRegistryResp {
			bu.commonRec.CheckAndCreateEventMessageObject(bu.ctx, resp, bu.bmcObj)
		}
	}

	if mgrDetails != nil {
		if firmwareVersion, ok := mgrDetails["FirmwareVersion"].(string); ok {
			bu.bmcObj.Status.FirmwareVersion = firmwareVersion
		}
	}
	storageDetails := bu.updateDriveDetails()
	if len(storageDetails) != 0 {
		bu.bmcObj.Status.StorageControllers = storageDetails
	} else {
		l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Could not successfully get list of drives for %s bmc", bu.bmcObj.Spec.BmcDetails.Address))
	}
	bu.commonRec.UpdateBmcStatus(bu.ctx, bu.bmcObj)
	common.MapOfBmc["/redfish/v1/Systems/"+sysId] = common.BmcState{IsAdded: false, IsDeleted: false, IsObjCreated: true}
	return true
}

// prepareBody will prepare the body for add bmc
func (bu *bmcUtils) PrepareAddBmcPayload(connectionMeth string) ([]byte, error) {
	connMeth := map[string]string{"@odata.id": connectionMeth}
	link := links{ConnectionMethod: connMeth}
	details := addBmcPayloadBody{HostName: bu.bmcObj.Spec.BmcDetails.Address, UserName: bu.bmcObj.Spec.Credentials.Username, Password: bu.bmcObj.Spec.Credentials.Password, Links: link}
	body, err := json.Marshal(details)
	if err != nil {
		l.LogWithFields(bu.ctx).Error(fmt.Sprintf("Error marshalling add request body for %s BMC: %s", bu.bmcObj.Spec.BmcDetails.Address, err.Error()))
		return nil, err
	}
	return body, nil
}

func (bu *bmcUtils) updateBmcLabels() {
	bu.bmcObj.ObjectMeta.Labels = map[string]string{}
	bu.bmcObj.ObjectMeta.Labels["name"] = bu.bmcObj.Spec.BmcDetails.Address
	bu.bmcObj.ObjectMeta.Labels["systemId"] = bu.bmcObj.Status.BmcSystemID
	bu.bmcObj.ObjectMeta.Labels["serialNo"] = bu.bmcObj.Status.SerialNumber
	bu.bmcObj.ObjectMeta.Labels["vendor"] = bu.bmcObj.Status.VendorName
	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		l.LogWithFields(bu.ctx).Error("Could not compile regex : " + err.Error())
	}
	bu.bmcObj.ObjectMeta.Labels["modelId"] = re.ReplaceAllString(bu.bmcObj.Status.ModelID, ".")
	bu.bmcObj.ObjectMeta.Labels["firmwareVersion"] = re.ReplaceAllString(bu.bmcObj.Status.FirmwareVersion, ".")
}

func (bu *bmcUtils) updateDriveDetails() map[string]infraiov1.ArrayControllers {
	storageControllers := map[string]infraiov1.ArrayControllers{}
	drives := map[string]infraiov1.DriveDetails{}
	var arrayController infraiov1.ArrayControllers
	raidLevels := []string{}
	storageDetails, _, err := bu.bmcRestClient.Get(fmt.Sprintf("/redfish/v1/Systems/%s/Storage/", bu.bmcObj.Status.BmcSystemID), fmt.Sprintf("Fetching Storage details for %s BMC..", bu.bmcObj.Spec.BmcDetails.Address))
	if err != nil {
		l.LogWithFields(bu.ctx).Error(err, "Error getting storage details")
		return storageControllers
	}
	if storageDetails["Members"] == nil {
		l.LogWithFields(bu.ctx).Info("No storage controllers present..")
		return storageControllers
	} else {
		arrContrDetails := storageDetails["Members"].([]interface{})
		for _, ac := range arrContrDetails {
			raidLevels = []string{}
			arrContlink := ac.(map[string]interface{})["@odata.id"].(string)
			arrContDriveLinks, _, err := bu.bmcRestClient.Get(arrContlink, fmt.Sprintf("Fetching array controller details for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
			if err != nil {
				l.LogWithFields(bu.ctx).Error(err, "Error getting array controllers")
				return storageControllers
			}
			if _, ok := arrContDriveLinks["Id"].(string); ok {
				arrContID := arrContDriveLinks["Id"].(string)
				if len(arrContDriveLinks["Drives"].([]interface{})) == 0 {
					l.LogWithFields(bu.ctx).Info(fmt.Sprintf("No drives present for %s", arrContID))
					arrayController.Drives = nil
				} else {
					drivesMap := arrContDriveLinks["Drives"].([]interface{})
					for _, driveLink := range drivesMap {
						drive := driveLink.(map[string]interface{})
						volumes := []int{}
						driveDetails, _, err := bu.bmcRestClient.Get(drive["@odata.id"].(string), fmt.Sprintf("Fetching drive details for %s BMC..", bu.bmcObj.Spec.BmcDetails.Address))
						if err != nil {
							l.LogWithFields(bu.ctx).Error(err, "Error getting drive details..")
							return storageControllers
						}
						driveID := driveDetails["Id"].(string)
						capacityBytes := driveDetails["CapacityBytes"].(float64)
						if len(driveDetails["Links"].(map[string]interface{})) == 0 {
							l.LogWithFields(bu.ctx).Info(fmt.Sprintf("No volumes are used for %s drive", driveID))
						} else {
							volumeLinks := driveDetails["Links"].(map[string]interface{})["Volumes"].([]interface{})
							for _, vl := range volumeLinks {
								volLink := vl.(map[string]interface{})["@odata.id"].(string)
								volID, _ := strconv.Atoi(volLink[len(volLink)-1:])
								volumes = append(volumes, volID)
							}
						}
						driveDet := infraiov1.DriveDetails{CapacityBytes: fmt.Sprintf("%f", capacityBytes), UsedInVolumes: volumes}
						drives[driveID] = driveDet
					}
					arrayController.Drives = drives
				}
				if len(arrContDriveLinks["Volumes"].(map[string]interface{})) == 0 {
					l.LogWithFields(bu.ctx).Info(fmt.Sprintf("No volumes are created for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
				} else {
					volumeLink := arrContDriveLinks["Volumes"].(map[string]interface{})["@odata.id"].(string)
					volumeDetails, sCode, err := bu.bmcRestClient.Get(volumeLink, fmt.Sprintf("Fetching volume details for %s BMC..", bu.bmcObj.Spec.BmcDetails.Address))
					if err != nil {
						l.LogWithFields(bu.ctx).Error(err, "Error getting volume details..")
						return storageControllers
					} else if sCode == http.StatusOK {
						var capabilityDetails []interface{}
						if collectionCap, ok := volumeDetails["@Redfish.CollectionCapabilities"]; ok {
							capabilityDetails = collectionCap.(map[string]interface{})["Capabilities"].([]interface{})
						}
						var capLink string
						for _, capObj := range capabilityDetails {
							capLink = capObj.(map[string]interface{})["CapabilitiesObject"].(map[string]interface{})["@odata.id"].(string)
							break
						}
						if capLink != "" {
							capDet, sCode, err := bu.bmcRestClient.Get(capLink, fmt.Sprintf("Fetching capability details for %s BMC..", bu.bmcObj.Spec.BmcDetails.Address))
							if err != nil {
								l.LogWithFields(bu.ctx).Error(err, "Error getting capability details..")
								return storageControllers
							} else if sCode == http.StatusOK {
								rl := capDet["RAIDType@Redfish.AllowableValues"].([]interface{})
								for _, raidType := range rl {
									raidLevels = append(raidLevels, raidType.(string))
								}
							} else {
								l.LogWithFields(bu.ctx).Info("Unable to get capability details, Try again..")
							}
						} else {
							l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Capability Object not found for %s volume", arrContID))
						}
					} else {
						l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Unable to get volume details for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
					}
				}
				arrayController.SupportedRAIDLevel = raidLevels
				storageControllers[arrContID] = arrayController
			}
		}
	}
	return storageControllers
}

// GetManagerDetails used to get manager response
func (bu *bmcUtils) GetManagerDetails() map[string]interface{} {
	systemUUID := strings.Split(bu.bmcObj.Status.BmcSystemID, ".")[0]
	resp, sCode, err := bu.bmcRestClient.Get("/redfish/v1/Managers/", fmt.Sprintf("Fetching managers collection details"))
	if err != nil {
		l.LogWithFields(bu.ctx).Errorf("Failed to get managers collection details with error: %v", err)
	}
	if sCode == http.StatusOK {
		members := resp["Members"].([]interface{})
		if len(members) > 0 {
			for _, member := range members {
				managerURI := member.(map[string]interface{})["@odata.id"].(string)
				uri := strings.Split(managerURI, "/")
				systemID := uri[len(uri)-1]
				managerSysUUID := strings.Split(systemID, ".")[0]
				if systemUUID == managerSysUUID {
					resp, sCode, err = bu.bmcRestClient.Get(managerURI, fmt.Sprintf("Fetching manager details for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
					if sCode == http.StatusOK {
						return resp
					}
				}
			}
		}
	}
	l.LogWithFields(bu.ctx).Error(fmt.Sprintf("Failed getting manager details for %s BMC: Got %d from odim: %s", bu.bmcObj.Spec.BmcDetails.Address, sCode, err.Error()))
	return nil
}

// ---------Delete BMC--------------
// unregisterBmc used to unregister BMC
func (bu *bmcUtils) unregisterBmc() bool {
	deleteURI := fmt.Sprintf("/redfish/v1/AggregationService/AggregationSources/%s", bu.bmcObj.Status.BmcSystemID)
	l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Deleting %s BMC from odim", bu.bmcObj.Spec.BmcDetails.Address))
	ok := bu.commonUtil.BmcDeleteOperation(bu.ctx, deleteURI, bu.bmcRestClient, bu.bmcObj.ObjectMeta.Name)
	if !ok {
		l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Error in getting delete response for BMC %s", bu.bmcObj.Spec.BmcDetails.Address))
		return false
	}
	return true
}

// loggingBmcDeletionActivity will log deletion activity
func (bu *bmcUtils) loggingBmcDeletionActivity(isDeleted error, objectName string) {
	if isDeleted != nil {
		l.LogWithFields(bu.ctx).Errorf("Error while deleting %s object for %s BMC!: %s", objectName, bu.bmcObj.Spec.BmcDetails.Address, isDeleted.Error())
	} else {
		l.LogWithFields(bu.ctx).Info(fmt.Sprintf("%s object deleted for %s BMC", objectName, bu.bmcObj.Spec.BmcDetails.Address))
	}
}

func (bu *bmcUtils) removeBmcFinalizerAndDeleteDependencies() {
	controllerutil.RemoveFinalizer(bu.bmcObj, bmcFinalizer)
	biosObj := bu.commonRec.GetBiosObject(bu.ctx, constants.MetadataName, bu.bmcObj.ObjectMeta.Name, bu.namespace)
	bootObj := bu.commonRec.GetBootObject(bu.ctx, constants.MetadataName, bu.bmcObj.ObjectMeta.Name, bu.namespace)
	volObj := bu.commonRec.GetVolumeObject(bu.ctx, bu.bmcObj.Spec.BmcDetails.Address, bu.namespace)
	firmwareObj := bu.commonRec.GetFirmwareObject(bu.ctx, constants.MetadataName, bu.bmcObj.ObjectMeta.Name, bu.namespace)
	isBiosDeleted := bu.commonRec.GetCommonReconcilerClient().Delete(bu.ctx, biosObj)
	bu.loggingBmcDeletionActivity(isBiosDeleted, "BiosSetting")
	isBootDeleted := bu.commonRec.GetCommonReconcilerClient().Delete(bu.ctx, bootObj)
	bu.loggingBmcDeletionActivity(isBootDeleted, "BootOrderSetting")
	isFirmwareDeleted := bu.commonRec.GetCommonReconcilerClient().Delete(bu.ctx, firmwareObj)
	bu.loggingBmcDeletionActivity(isFirmwareDeleted, "Firmware")
	if volObj != nil {
		controllerutil.RemoveFinalizer(volObj, volFinalizer)
		err := bu.commonRec.GetCommonReconcilerClient().Update(bu.ctx, volObj)
		if err != nil {
			l.LogWithFields(bu.ctx).Errorf("Error while updating volume object for %s BMC!: %s", bu.bmcObj.Spec.BmcDetails.Address, err)
		}
		isVolumeDeleted := bu.commonRec.GetCommonReconcilerClient().Delete(bu.ctx, volObj)
		bu.loggingBmcDeletionActivity(isVolumeDeleted, "Volume")
	}
}

func (bu *bmcUtils) deleteBMCObject() {
	sysID := bu.bmcObj.Status.BmcSystemID
	err := bu.commonRec.GetCommonReconcilerClient().Delete(context.Background(), bu.bmcObj)
	if err != nil {
		l.LogWithFields(bu.ctx).Errorf("Error: Deleting the BMC object: %s", err.Error())
	}
	delete(common.MapOfBmc, "/redfish/v1/Systems/"+sysID)
}

// ---------------Reset BMC-----------------
// getAllowableResetValues used to get allowable reset values from map
func (bu *bmcUtils) getAllowableResetValues() []string {
	key := infraiov1.SystemDetail{
		Vendor:  bu.bmcObj.Status.VendorName,
		ModelID: bu.bmcObj.Status.ModelID,
	}
	if val, ok := infraiov1.AllowableResetValues[key]; ok {
		return val
	}
	return nil
}

// validateResetData used to validate the input given by user for reset operation
// validateResetData(currentPowerState, bmcObj.Spec.BmcDetails.PowerState, bmcObj.Spec.BmcDetails.ResetType, allowableVal,*bmcObj)
func (bu *bmcUtils) validateResetData(powerState string, allowableValues []string) bool {
	allowedVal := make(map[string]bool)
	powerState, desiredPowerState := strings.ToUpper(powerState), strings.ToUpper(bu.bmcObj.Spec.BmcDetails.PowerState)
	for _, av := range allowableValues {
		allowedVal[av] = true
	}
	if !allowedVal[bu.bmcObj.Spec.BmcDetails.ResetType] {
		l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Reset value `%s` is not allowed by %s BMC, current system state is  `%s`", bu.bmcObj.Spec.BmcDetails.ResetType, bu.bmcObj.Spec.BmcDetails.Address, powerState))
		return false
	}
	state := infraiov1.State{PowerState: powerState, DesiredPowerState: desiredPowerState, ResetType: bu.bmcObj.Spec.BmcDetails.ResetType}
	if ResetWhenOnOn, ok := infraiov1.ResetWhenOnOn[state]; ok {
		if !ResetWhenOnOn {
			l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Reset with value `%s` is not allowed by %s BMC, current system state is  `%s`", bu.bmcObj.Spec.BmcDetails.ResetType, bu.bmcObj.Spec.BmcDetails.Address, powerState))
			return false
		}
		return true
	}
	if ResetWhenOffOff, ok := infraiov1.ResetWhenOffOff[state]; ok {
		if !ResetWhenOffOff {
			l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Reset with value `%s` is not allowed for %s BMC", bu.bmcObj.Spec.BmcDetails.ResetType, bu.bmcObj.Spec.BmcDetails.Address))
			return false
		}
		return true
	}
	if ResetWhenOnOff, ok := infraiov1.ResetWhenOnOff[state]; ok {
		if !ResetWhenOnOff {
			l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Reset with value `%s` is not allowed for %s BMC", bu.bmcObj.Spec.BmcDetails.ResetType, bu.bmcObj.Spec.BmcDetails.Address))
			return false
		}
		return true
	}
	if ResetWhenOffOn, ok := infraiov1.ResetWhenOffOn[state]; ok {
		if !ResetWhenOffOn {
			l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Reset with value `%s` is not allowed for %s BMC", bu.bmcObj.Spec.BmcDetails.ResetType, bu.bmcObj.Spec.BmcDetails.Address))
			return false
		}
		return true
	}

	return false
}

// resetSystem used to reset computer system
func (bu *bmcUtils) ResetSystem(isBiosUpdation bool, updateBmcDependents bool) bool {
	l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Resetting system with reset type as: %s", bu.bmcObj.Spec.BmcDetails.ResetType))
	uri := "/redfish/v1/Systems/" + bu.bmcObj.Status.BmcSystemID + "/Actions/ComputerSystem.Reset"
	reqBody := &common.ComputerSystemReset{
		ResetType: bu.bmcObj.Spec.BmcDetails.ResetType,
	}
	body, jsonerr := json.Marshal(reqBody)
	if jsonerr != nil {
		l.LogWithFields(bu.ctx).Error(fmt.Sprintf("Error marshalling reset payload for %s BMC: %s", bu.bmcObj.Spec.BmcDetails.Address, jsonerr.Error()))
		return false
	}
	response, err := bu.bmcRestClient.Post(uri, fmt.Sprintf("Posting reset payload to %s BMC", bu.bmcObj.Spec.BmcDetails.Address), body)
	if err != nil || response.StatusCode != http.StatusAccepted {
		l.LogWithFields(bu.ctx).Error(fmt.Sprintf("Error while resetting %s BMC: %s: %s", bu.bmcObj.Spec.BmcDetails.Address, strconv.Itoa(response.StatusCode), err.Error()))
		return false
	}
	if response.StatusCode == http.StatusUnauthorized {
		l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Password is not authorised for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
		return false
	}
	done, _ := bu.commonUtil.MoniteringTaskmon(response.Header, bu.ctx, common.RESETBMC, bu.bmcObj.ObjectMeta.Name)
	if done {
		if updateBmcDependents { // this is introduced to skip updating of bmc dependents when revert of volume deleted scenario takes place.
			biosObj := bu.commonRec.GetBiosObject(bu.ctx, constants.MetadataName, bu.bmcObj.ObjectMeta.Name, bu.namespace)
			biosUtil := bios.GetBiosUtils(bu.ctx, biosObj, bu.commonRec, bu.bmcRestClient, bu.namespace)
			biosAttribute := getUpdatedBiosAttributes(bu.ctx, biosObj.Status.BiosAttributes, bu.bmcObj, biosUtil)
			biosUtil.UpdateBiosAttributesOnReset(bu.bmcObj.Spec.BmcDetails.Address, biosAttribute)
			bootObj := bu.commonRec.GetBootObject(bu.ctx, constants.MetadataName, bu.bmcObj.ObjectMeta.Name, bu.namespace)
			oldBootAttribute := bootObj.Status.Boot
			bootUtil := boot.GetBootUtils(bu.ctx, nil, bu.commonRec, bu.bmcRestClient, bu.commonUtil, bu.namespace)
			for i := 0; i < 10; i++ {
				sysDetails := bu.commonUtil.GetBmcSystemDetails(bu.ctx, bu.bmcObj)
				if sysDetails != nil {
					bootAttribute := bootUtil.GetBootAttributes(sysDetails)
					if bootAttribute != nil && !reflect.DeepEqual(bootAttribute, oldBootAttribute) {
						bootUtil.UpdateBootAttributesOnReset(bu.bmcObj.ObjectMeta.Name, bootAttribute)
						break
					}
				}
				time.Sleep(time.Duration(constants.SleepTime) * time.Second)
			}
			if strings.Contains(bu.bmcObj.Status.SystemReset, "Volume") {
				strArr := strings.Split(bu.bmcObj.Status.SystemReset, " ")
				volName := strArr[len(strArr)-2]
				volumeObj := bu.commonRec.GetVolumeObject(bu.ctx, bu.bmcObj.Spec.BmcDetails.Address, bu.namespace)
				volUtil := volume.GetVolumeUtils(bu.ctx, bu.bmcRestClient, bu.commonRec, volumeObj, bu.namespace)
				volUpdated, updateMsg := volUtil.UpdateVolumeStatusAndClearSpec(bu.bmcObj, volName)
				if !volUpdated {
					l.LogWithFields(bu.ctx).Info(updateMsg)
				}
			}

		}
		delete(common.RestartRequired, bu.bmcObj.Status.BmcSystemID)
		return true
	}
	return false
}

func getUpdatedBiosAttributes(ctx context.Context, oldBiosAttributes map[string]string, bmcObj *infraiov1.Bmc, biosUtil bios.BiosInterface) map[string]string {
	var updatedBiosAttributes map[string]string
	for i := 0; i < 10; i++ {
		biosAttribute := biosUtil.GetBiosAttributes(bmcObj)
		if biosAttribute != nil && !reflect.DeepEqual(biosAttribute, oldBiosAttributes) {
			updatedBiosAttributes = biosAttribute
			break
		}
		time.Sleep(time.Duration(constants.SleepTime) * time.Second)
	}
	return updatedBiosAttributes
}

// ----------Update New Password on BMC,ODIM----------------
// updateOdimWithNewPassword will update ODIM with new working password of bmc
func (bu *bmcUtils) updateOdimWithNewPassword() (bool, error) {
	aggUri := fmt.Sprintf("/redfish/v1/AggregationService/AggregationSources/%s", bu.bmcObj.Status.BmcSystemID)
	l.LogWithFields(bu.ctx).Info("Updating password on ODIM")
	bdy := modifyPassword{Password: bu.bmcObj.Spec.Credentials.Password}
	body, err := json.Marshal(bdy)
	if err != nil {
		l.LogWithFields(bu.ctx).Error("Error marshaling request payload to ODIM" + err.Error())
		return false, err
	}
	patchResp, err := bu.bmcRestClient.Patch(aggUri, fmt.Sprintf("Patching ODIM with new password for %s BMC..", bu.bmcObj.Spec.BmcDetails.Address), body)
	if err == nil && patchResp.StatusCode == http.StatusOK {
		return true, nil
	}
	if patchResp.StatusCode == http.StatusUnauthorized {
		l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Password is not authorised for %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
		return false, nil
	}
	return false, err
}

// updateBmc will update the bmc with new user entered password
func (bu *bmcUtils) updateBmcWithNewPassword() (bool, error) {
	remoteAccUri := fmt.Sprintf("/redfish/v1/Managers/%s/RemoteAccountService/Accounts", bu.bmcObj.Status.BmcSystemID)
	l.LogWithFields(bu.ctx).Info(fmt.Sprintf("Updating password on %s BMC", bu.bmcObj.Spec.BmcDetails.Address))
	var changePassUri string
	getResp, _, _ := bu.bmcRestClient.Get(remoteAccUri, "Fetching remote account members..")
	accounts := getResp["Members"].([]interface{})
	for _, acc := range accounts {
		accUri := acc.(map[string]interface{})["@odata.id"].(string)
		accGet, _, _ := bu.bmcRestClient.Get(accUri, "Fetching remote account details..")
		if accGet["UserName"].(string) == bu.bmcObj.Spec.Credentials.Username {
			changePassUri = accUri
			break
		}
	}
	bdy := modifyPassword{Password: bu.bmcObj.Spec.Credentials.Password}
	body, err := json.Marshal(bdy)
	if err != nil {
		l.LogWithFields(bu.ctx).Error("Error marshaling request payload to BMC: " + err.Error())
		return false, err
	}
	patchResp, err := bu.bmcRestClient.Patch(changePassUri, fmt.Sprintf("Patching new password on %s BMC", bu.bmcObj.Spec.BmcDetails.Address), body)
	if err != nil {
		return false, err
	}
	done, _ := bu.commonUtil.MoniteringTaskmon(patchResp.Header, bu.ctx, common.UPDATEBMC, bu.bmcObj.ObjectMeta.Name)
	if done {
		return true, nil
	}
	return false, err
}

// -------------Utils-------------------
// encryptPassword will encrypt the current password of the bmc
func (bu *bmcUtils) encryptPassword(publicKey *rsa.PublicKey) {
	encryptedPassword := utils.EncryptWithPublicKey(bu.ctx, bu.bmcObj.Spec.Credentials.Password, *utils.PublicKey)
	if encryptedPassword != "" {
		bu.bmcObj.Spec.Credentials.Password = encryptedPassword
		bu.bmcObj.ObjectMeta.Annotations["old_password"] = encryptedPassword
	}
	l.LogWithFields(bu.ctx).Info("Could not encrypt password")
}

// GetConnectionMethod is used to get connection method
func (bu *bmcUtils) GetConnectionMethod(odimObj *infraiov1.Odim) string {
	connMethInfo := odimObj.Status.ConnMethVariants
	for connMethVar, connMeth := range connMethInfo {
		if bu.bmcObj.Spec.BmcDetails.ConnMethVariant == connMethVar {
			return connMeth
		}
	}
	return ""
}

func (bu *bmcUtils) UpdateBmcObject(bmcObj *infraiov1.Bmc) {
	bu.bmcObj = bmcObj
}

func GetBmcUtils(ctx context.Context, bmcObj *infraiov1.Bmc, ns string, commonRec *utils.ReconcilerInterface, bmcRestClient *restclient.RestClientInterface, commonUtil common.CommonInterface, updateBmcObject bool) BmcInterface {
	return &bmcUtils{
		ctx:             ctx,
		bmcObj:          bmcObj,
		namespace:       ns,
		commonRec:       *commonRec,
		bmcRestClient:   *bmcRestClient,
		commonUtil:      commonUtil,
		updateBMCObject: updateBmcObject,
	}
}

// TODO: Will be used to get reset allowable values in future

/*
var AllowableResetValues = make(map[string]interface{})
var vendor = []string{"HPE", "DELL", "LENOVO"}
// getAllowableResetValues used to check and store allowable reset values of added bmc in map
func getAllowableResetValues() {
	uri := "/redfish/v1/Systems/" + bmcObj.Status.BmcSystemID
	resp, sCode, _ := odimClient.Get(uri)
	if sCode == http.StatusOK {
		if resp["Manufacturer"] == nil || resp["Model"] == nil {
			l.Log.Info("Vendor and model details missing")
			return
		if resp["PowerState"].(string) != bmcObj.Spec.BmcDetails.PowerState {
			l.Log.Info(fmt.Sprintf("current power state %s", resp["PowerState"].(string)))
			resetSystem(bmcObj.Spec.BmcDetails.PowerState, bmcObj.Status.BmcSystemID)
			bmcObj.Spec.BmcDetails.PowerState = resp["PowerState"].(string)
		}
		if substr, ok := CaseInsensitiveContains(resp["Manufacturer"].(string)); ok {
			key := substr + "-" + resp["Model"].(string)
			if _, ok := AllowableResetValues[key]; !ok {
				if resp["Actions"] != nil || resp["Actions"].(map[string]interface{})["#ComputerSystem.Reset"] != nil {
					val := resp["Actions"].(map[string]interface{})["#ComputerSystem.Reset"].(map[string]interface{})["ResetType@Redfish.AllowableValues"].([]interface{})
					AllowableResetValues[key] = val
					l.Log.Info(fmt.Sprintf("allowable values added to map with key as %s", key))
				}
			}
		} else {
			l.Log.Info(fmt.Sprintf("%s vendor is not valid", resp["Manufacturer"].(string)))
		}
	}
	l.Log.Info(fmt.Sprintf("Map of allowable values %s", AllowableResetValues))
}
func CaseInsensitiveContains(s string) (string, bool) {
	for _, substr := range vendor {
		s, substr = strings.ToUpper(s), strings.ToUpper(substr)
		if strings.Contains(s, substr) {
			return substr, true
		}
	}
	return "", false
} */
