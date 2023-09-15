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
	"strconv"
	"strings"

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

// BiosSettingReconciler reconciles a BiosSetting object
type BiosSettingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	biosSchemaObject *infraiov1.BiosSchemaRegistry
)

// AttributeEnumValue struct defines list of attribute values
type AttributeEnumValue struct {
	ValueDisplayName string `json:"ValueDisplayName"`
	ValueName        string `json:"ValueName"`
}

var podName = os.Getenv("POD_NAME")

//+kubebuilder:rbac:groups=infra.io.odimra,resources=biossettings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=biossettings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=biossettings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BiosSetting object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *BiosSettingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	transactionId := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionId.String(), constants.BmcOperator, constants.BIOSSettingActionID, constants.BIOSSettingActionName, podName)
	//common reconciler ddeclaration
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	//create rest client
	odimObj := commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", req.Namespace)
	biosRestClient, err := restclient.NewRestClient(ctx, odimObj, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get rest client for Bios: %s", err.Error())
		return ctrl.Result{}, err
	}
	biosObj := &infraiov1.BiosSetting{}
	// Fetch the Bios instance.
	err = r.Get(ctx, req.NamespacedName, biosObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// check to skip reconcilation from running while bisosetting object is created when bmc is added
	if len(biosObj.Spec.Bios) == 0 {
		return ctrl.Result{}, nil
	}
	biosUtil := GetBiosUtils(ctx, biosObj, commonRec, biosRestClient, req.Namespace)
	// if user applies bios.yaml
	// Check if any one of the details is present and properties is not empty
	if biosObj.Spec.BmcName != "" || biosObj.Spec.SerialNo != "" || biosObj.Spec.SystemID != "" {
		//get system URI
		// var systemURI, systemID, biosID string
		var bmcObject *infraiov1.Bmc
		if biosObj.Spec.SystemID != "" {
			bmcObject = commonRec.GetBmcObject(ctx, constants.StatusBmcSystemID, biosObj.Spec.SystemID, req.Namespace)
			if bmcObject == nil {
				l.LogWithFields(ctx).Info("Check if BMC is registered..")
				return ctrl.Result{}, nil
			}
		} else if biosObj.Spec.BmcName != "" {
			bmcObject = commonRec.GetBmcObject(ctx, constants.SpecBmcAddress, biosObj.Spec.BmcName, req.Namespace)
			if bmcObject == nil {
				l.LogWithFields(ctx).Info("Check if BMC is registered..")
				return ctrl.Result{}, nil
			}
		} else if biosObj.Spec.SerialNo != "" {
			bmcObject = commonRec.GetBmcObject(ctx, constants.StatusSerialNumber, biosObj.Spec.SerialNo, req.Namespace)
			if bmcObject.Status.SerialNumber == "" {
				l.LogWithFields(ctx).Info("Check if BMC is registered..")
				return ctrl.Result{}, nil
			}
		}
		biosBmcIP := bmcObject.Spec.BmcDetails.Address
		systemID := bmcObject.Status.BmcSystemID
		biosID := bmcObject.Status.BiosAttributeRegistry
		if systemID == "" {
			return ctrl.Result{}, nil
		}
		systemURI := fmt.Sprintf("/redfish/v1/Systems/%s", systemID)
		systemsGetResp, _, err := biosRestClient.Get(systemURI, "Getting response on systems")
		if err != nil {
			l.LogWithFields(ctx).Errorf("Error on GET: %s, try again : %s", systemURI, err.Error())
			return ctrl.Result{}, nil
		}
		ok, body := biosUtil.ValidateBiosAttributes(biosID)
		if !ok {
			return ctrl.Result{}, nil
		}
		if systemsGetResp["Bios"] != nil {
			//get bios url
			biosLink, biosBody := biosUtil.getBiosLinkAndBody(systemsGetResp, body, biosBmcIP)
			if biosBody != nil && biosLink != "" {
				//send patch req on bios url
				patchResponse, err := biosRestClient.Patch(biosLink, fmt.Sprintf("Patching bios payload for %s BMC", biosBmcIP), biosBody)
				if err != nil {
					l.LogWithFields(ctx).Error(fmt.Sprintf("error while patching bios for %s BMC: ", bmcObject.Spec.BmcDetails.Address), err.Error())
					return ctrl.Result{}, nil
				}
				if patchResponse.StatusCode == http.StatusAccepted {
					//add taskmon here
					done, _ := biosUtil.(*biosUtils).commonUtil.MoniteringTaskmon(patchResponse.Header, ctx, common.BIOSSETTING, bmcObject.ObjectMeta.Name)
					if done {
						l.LogWithFields(ctx).Info("Bios configured, Please reset system now.")
						commonRec.UpdateBmcObjectOnReset(ctx, bmcObject, fmt.Sprintf("%s Bios", constants.PendingForResetEvent))
						return ctrl.Result{}, nil
					} else {
						err = commonRec.GetCommonReconcilerClient().Delete(ctx, biosObj)
						if err != nil {
							l.LogWithFields(ctx).Error(fmt.Sprintf("error while deleteing bios setting object for %s BMC:", biosBmcIP), err.Error())
						}
					}
				} else {
					l.LogWithFields(ctx).Info(fmt.Sprintf("Could not update bios settings for %s BMC, try again!", bmcObject.Spec.BmcDetails.Address))
					err = r.Delete(ctx, biosObj)
					if err != nil {
						l.LogWithFields(ctx).Error("error while deleting bios object" + err.Error())
					}
					return ctrl.Result{}, nil
				}
				l.LogWithFields(ctx).Errorf("Error in patching, bios not configured properly for %s BMC, try again : %s", biosBmcIP, err.Error())
			} else {
				l.LogWithFields(ctx).Info("Unable to configure bios settings, Please try again.")
			}
		} else {
			l.LogWithFields(ctx).Infof("Couldn't get bios details on %s, Please check SystemID again", systemURI)
		}
	} else {
		l.LogWithFields(ctx).Info("Please enter either of BmcIP/SerialNo/SystemID")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BiosSettingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.BiosSetting{}).
		WithEventFilter(utils.IgnoreStatusUpdate()).
		Complete(r)
}

// ValidateBiosAttributes is used to validate the input data for bios attributes.
func (bs *biosUtils) ValidateBiosAttributes(biosID string) (bool, map[string]interface{}) {
	l.LogWithFields(bs.ctx).Info("Validating BIOS Attributes..")
	schemaName := utils.RemoveSpecialChar(biosID)
	biosSchemaObject = bs.commonRec.GetBiosSchemaObject(bs.ctx, constants.MetadataName, schemaName, bs.namespace)
	resBody := make(map[string]interface{})
	if biosSchemaObject.Spec.Attributes == nil {
		l.LogWithFields(bs.ctx).Errorf("No object with schema name %s present!", schemaName)
		return false, nil
	}
	attributeColl := biosSchemaObject.Spec.Attributes
	for attr, value := range bs.biosObj.Spec.Bios {
		for _, val := range attributeColl {
			if val["AttributeName"] == attr {
				if val["ReadOnly"] == "true" {
					continue
				}
				if val["Type"] == "String" {
					i, _ := strconv.Atoi(val["MaxLength"])
					if len(value) > i {
						l.LogWithFields(bs.ctx).Error(fmt.Sprintf("Attribute %s max length size exceeded, required is %s", attr, val["MaxLength"]))
						return false, nil
					}
					resBody[attr] = value
				}
				if val["Type"] == "Integer" {
					i, err := strconv.Atoi(value)
					if err != nil {
						l.LogWithFields(bs.ctx).Error(fmt.Sprintf("Attribute %s should be of type Integer %s: %s", attr, value, err.Error()))
						return false, nil
					}
					lowerBound, _ := strconv.Atoi(val["LowerBound"])
					upperBound, _ := strconv.Atoi(val["UpperBound"])
					if i < lowerBound || i > upperBound {
						l.LogWithFields(bs.ctx).Error(fmt.Sprintf("Attribute %s should be within the bound limit", attr))
						return false, nil
					}
					resBody[attr] = i
				}
				if val["Type"] == "Enumeration" {
					var mapData []AttributeEnumValue
					var isPresent bool
					res := val["Value"]
					json.Unmarshal([]byte(res), &mapData)
					for _, mapValue := range mapData {
						if value == mapValue.ValueName {
							isPresent = true
						}
					}
					if !isPresent {
						l.LogWithFields(bs.ctx).Errorf("Invalid attribute value %s", value)
						return false, nil
					}
					resBody[attr] = value
				}
			}
		}
	}
	return true, resBody
}

// getBiosLinkAndBody fetches the bios uri and builds the bios body
func (bs *biosUtils) getBiosLinkAndBody(resp map[string]interface{}, biosProps map[string]interface{}, biosBmcIP string) (string, []byte) {
	biosId := resp["Bios"].(map[string]interface{})["@odata.id"].(string)
	bs.biosObj.ObjectMeta.Annotations["odata.id"] = biosId
	err := bs.commonRec.GetCommonReconcilerClient().Update(bs.ctx, bs.biosObj)
	if err != nil {
		l.LogWithFields(bs.ctx).Error(fmt.Sprintf("Error: Updating %s BIOS object annotations of bmc: %s", biosBmcIP, err.Error()))
	}
	biosResp, _, err := bs.biosRestClient.Get(biosId, fmt.Sprintf("Fetching Bios Details for %s BMC", biosBmcIP))
	if err != nil {
		l.LogWithFields(bs.ctx).Errorf("Error fetching on Bios odata.id of %s bmc: %s", biosBmcIP, err.Error())
		return "", nil
	}
	biosLink := biosResp["@Redfish.Settings"].(map[string]interface{})["SettingsObject"].(map[string]interface{})["@odata.id"].(string)
	bios_settings := map[string]map[string]interface{}{"Attributes": biosProps}
	biosBody, err := json.Marshal(bios_settings)
	if err != nil {
		l.LogWithFields(bs.ctx).Errorf("Failed to marshal Bios Payload: %s", err.Error())
		biosBody = nil
	}
	return biosLink, biosBody
}

// updateBiosAttributesOnReset updates the bios attributes with latest bios values
func (bs *biosUtils) UpdateBiosAttributesOnReset(biosBmcIP string, updatedBiosAttributes map[string]string) {
	bs.biosObj.Status.BiosAttributes = updatedBiosAttributes
	bs.biosObj.Spec.Bios = map[string]string{}
	err := bs.commonRec.GetCommonReconcilerClient().Update(bs.ctx, bs.biosObj)
	if err != nil {
		l.LogWithFields(bs.ctx).Errorf("Error: Updating Spec of Bios Setting of %s: %s", biosBmcIP, err.Error())
	}
	err = bs.commonRec.GetCommonReconcilerClient().Update(bs.ctx, bs.biosObj)
	if err != nil {
		l.LogWithFields(bs.ctx).Errorf("Error: Updating Status of Bios Setting of %s: %s", biosBmcIP, err.Error())
	}
}

// getBiosAttributes is used to get bios attributes for added system
func (bs *biosUtils) GetBiosAttributes(bmcObj *infraiov1.Bmc) map[string]string {
	biosAttributesMap := make(map[string]string)
	uri := "/redfish/v1/Systems/" + bmcObj.Status.BmcSystemID + "/Bios"
	resp, sCode, err := bs.biosRestClient.Get(uri, fmt.Sprintf("Fetching BIOS details for %s BMC", bmcObj.Spec.BmcDetails.Address))
	if err != nil {
		l.LogWithFields(bs.ctx).Errorf("failed to fetch BIOS attribute details: %v", err)
		return nil
	}
	if sCode != http.StatusOK {
		l.LogWithFields(bs.ctx).Errorf("failed to fetch BIOS attribute details for BMC: got status %d", sCode)
		return nil
	}
	attributes, ok := resp["Attributes"].(map[string]interface{})
	if !ok {
		l.LogWithFields(bs.ctx).Errorf("failed to parse BIOS attributes from response")
		return nil
	}
	for key, attributeVal := range attributes {
		biosAttributesMap[key] = fmt.Sprintf("%v", attributeVal)
	}
	return biosAttributesMap
}

// GetBiosAttributeID used to get bios attribute ID
func (bs *biosUtils) GetBiosAttributeID(biosVersion string, bmcObj *infraiov1.Bmc) (string, map[string]interface{}) {
	uri := "/redfish/v1/Registries/"
	resp, sCode, _ := bs.biosRestClient.Get(uri, fmt.Sprintf("Fetching registries details for %s BMC", bmcObj.Spec.BmcDetails.Address))
	if sCode == http.StatusOK {
		for _, member := range resp["Members"].([]interface{}) {
			val := member.(map[string]interface{})["@odata.id"].(string)
			if strings.Contains(val, "BiosAttributeRegistry") {
				res, statusCode, _ := bs.biosRestClient.Get(val, fmt.Sprintf("Fetching bios attribute registry details for %s BMC", bmcObj.Spec.BmcDetails.Address))
				if statusCode == http.StatusOK {
					for _, mem := range res["Location"].([]interface{}) {
						if mem.(map[string]interface{})["Language"].(string) == "en" {
							url := mem.(map[string]interface{})["Uri"].(string)
							r, s, _ := bs.biosRestClient.Get(url, fmt.Sprintf("Fetching bios attribute registry JSON for %s BMC", bmcObj.Spec.BmcDetails.Address))
							if s == http.StatusOK {
								for _, ss := range r["SupportedSystems"].([]interface{}) {
									if ss.(map[string]interface{})["FirmwareVersion"].(string) == biosVersion {
										return r["Id"].(string), r
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return "", nil
}

func GetBiosUtils(ctx context.Context, biosObj *infraiov1.BiosSetting, commonRec utils.ReconcilerInterface, biosRestClient restclient.RestClientInterface, ns string) BiosInterface {
	return &biosUtils{
		ctx:            ctx,
		biosObj:        biosObj,
		commonRec:      commonRec,
		biosRestClient: biosRestClient,
		commonUtil:     common.GetCommonUtils(biosRestClient),
		namespace:      ns,
	}
}
