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
	"os"
	"reflect"
	"strings"
	"time"

	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	"golang.org/x/exp/slices"
	Error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/google/uuid"
)

var podName = os.Getenv("POD_NAME")

// EventsubscriptionReconciler reconciles a Eventsubscription object
type EventsubscriptionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infra.io.odimra,resources=eventsubscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=eventsubscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=eventsubscriptions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Eventsubscription object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *EventsubscriptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	transactionID := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionID.String(), constants.BmcOperator, constants.EventSubscriptionActionID, constants.EventSubscriptionActionName, podName)
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	eventSubscriptionObj := &infraiov1.Eventsubscription{}
	err := r.Get(ctx, req.NamespacedName, eventSubscriptionObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	odimObject := commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", req.Namespace)

	eventSubRestClient, err := restclient.NewRestClient(ctx, odimObject, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get rest client for Eventsubscription: %s", err.Error())
		return ctrl.Result{}, err
	}

	//get eventsubscriptionUtil object
	eventSubscriptionUtil := GetEventSubscriptionUtils(ctx, eventSubRestClient, commonRec, eventSubscriptionObj, req.Namespace)

	//Checking for deletion
	isEventSubscriptionMarkedForDelete := eventSubscriptionObj.GetDeletionTimestamp() != nil
	if isEventSubscriptionMarkedForDelete {
		if controllerutil.ContainsFinalizer(eventSubscriptionObj, constants.EventsubscriptionFinalizer) {
			isEventSubscriptionDeleted := eventSubscriptionUtil.DeleteEventsubscription()
			if isEventSubscriptionDeleted {
				controllerutil.RemoveFinalizer(eventSubscriptionObj, constants.EventsubscriptionFinalizer)
				if err := r.Update(ctx, eventSubscriptionObj); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				l.LogWithFields(ctx).Info(fmt.Sprintf("EventSubscription with ID %s not deleted! Retrying..", eventSubscriptionObj.Status.ID))
				return ctrl.Result{Requeue: true}, nil
			}
		}
		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(eventSubscriptionObj.Spec, infraiov1.EventsubscriptionSpec{}) {
		return ctrl.Result{}, nil
	}

	if eventSubscriptionObj.Spec.Destination != "" && eventSubscriptionObj.Spec.Context != "" {

		if !controllerutil.ContainsFinalizer(eventSubscriptionObj, constants.EventsubscriptionFinalizer) {
			controllerutil.AddFinalizer(eventSubscriptionObj, constants.EventsubscriptionFinalizer)
			err := r.Client.Update(ctx, eventSubscriptionObj)
			if err != nil {
				l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating %s eventsunscription object: %s", eventSubscriptionObj.Status.Name, err.Error()))
				return ctrl.Result{}, err
			}
			time.Sleep(time.Duration(constants.SleepTime) * time.Second)
		}

		//create Eventsubscription
		eventSubCreated, eventErr := eventSubscriptionUtil.CreateEventsubscription(eventSubscriptionObj, req.NamespacedName)
		if !eventSubCreated {
			return ctrl.Result{}, eventErr
		}
		l.LogWithFields(ctx).Info("Updating labels for event subscription object..")
		eventSubscriptionUtil.UpdateEventSubscriptionLabels()
		l.LogWithFields(ctx).Info("Successfully updated lables for event subscription object")

	} else {
		l.LogWithFields(ctx).Info("Please enter all the required details for creating a event subscription")
	}

	return ctrl.Result{}, nil
}

func (esu *eventsubscriptionUtils) CreateEventsubscription(eventSubObj *infraiov1.Eventsubscription, namespacedName types.NamespacedName) (bool, error) {
	esu.commonRec.GetUpdatedEventsubscriptionObjects(esu.ctx, namespacedName, esu.eventSubObj)
	if esu.eventSubObj.Status.ID != "" {
		return true, nil
	}
	// eventsubscription creation request
	eventSubReq, err := esu.GetEventSubscriptionRequest()
	if err == nil {
		//send request for event subscription creation
		resp, err := esu.eventsubRestClient.Post("/redfish/v1/EventService/Subscriptions", "Posting eventsubscription creation payload...", eventSubReq)
		if err != nil {
			l.LogWithFields(esu.ctx).Error("error while creating eventsubscription: " + err.Error())
			return false, err
		}
		if resp.StatusCode == http.StatusAccepted {
			done, resp := esu.commonUtil.MoniteringTaskmon(resp.Header, esu.ctx, common.EVENTSUBSCRIPTION, esu.eventSubObj.ObjectMeta.Name)
			if done {
				l.LogWithFields(esu.ctx).Info("\nEventsubscription created successfully!\n")
				l.LogWithFields(esu.ctx).Info("Updating Eventsubscription status...")
				statusUpdated := esu.UpdateEventsubscriptionStatus("")
				if !statusUpdated {
					l.LogWithFields(esu.ctx).Info("Eventsubscription created successfully but failed to update the status")
				} else {
					l.LogWithFields(esu.ctx).Info("Eventsubscription status updated successfully!")
				}
				return true, nil
			} else {
				l.LogWithFields(esu.ctx).Errorf("\n\nEventsubscription creation failed with error: %v\n\n", resp)
			}
		}
	}

	controllerutil.RemoveFinalizer(esu.eventSubObj, constants.EventsubscriptionFinalizer)
	err = esu.commonRec.GetCommonReconcilerClient().Update(esu.ctx, esu.eventSubObj)
	if err != nil {
		l.LogWithFields(esu.ctx).Error("error while updating eventsubscription object for deletion:" + err.Error())
	}
	err = esu.commonRec.GetCommonReconcilerClient().Delete(esu.ctx, esu.eventSubObj)
	if err != nil {
		l.LogWithFields(esu.ctx).Error("error while deleting eventsubscription object:" + err.Error())
	}
	l.LogWithFields(esu.ctx).Info(fmt.Sprintf("Could not create %s eventsubscription, try again", esu.eventSubObj.ObjectMeta.Name))
	return false, nil
}

func (esu *eventsubscriptionUtils) UpdateEventsubscriptionStatus(eventSubscriptionID string) bool {

	subscriptionList, statusCode, err := esu.eventsubRestClient.Get("/redfish/v1/EventService/Subscriptions", "Fetching all subsciptions..")
	if err != nil {
		l.LogWithFields(esu.ctx).Error("error while fetching all subscriptions:" + err.Error())
		return false
	} else if statusCode != http.StatusOK {
		l.LogWithFields(esu.ctx).Info("Failed to get all the subscriptions, try again..")
		return false
	}

	if subscriptions, ok := subscriptionList["Members"].([]interface{}); ok {
		for _, subscription := range subscriptions {
			reqURL := subscription.(map[string]interface{})["@odata.id"].(string)

			subscriptionDetails, statusCode, err := esu.eventsubRestClient.Get(reqURL, fmt.Sprintf("Fetching %s eventsubscription details..", reqURL[len(reqURL)-1:]))

			if err != nil {
				l.LogWithFields(esu.ctx).Error(fmt.Sprintf("error while fetching %s eventsubscription details:", reqURL[len(reqURL)-1:]) + err.Error())
				return false
			} else if statusCode != http.StatusOK {
				l.LogWithFields(esu.ctx).Info(fmt.Sprintf("Failed to get %s eventsubscription details, try again.", reqURL[len(reqURL)-1:]))
				return false
			} else {

				if subscriptionDetails["Destination"].(string) == esu.eventSubObj.Spec.Destination && subscriptionDetails["Context"].(string) == esu.eventSubObj.Spec.Context {
					originResources, err := esu.MapOriginResources(subscriptionDetails)
					if err != nil {
						l.LogWithFields(esu.ctx).Error(err.Error())
					}
					esu.commonRec.UpdateEventsubscriptionStatus(esu.ctx, esu.eventSubObj, subscriptionDetails, originResources)
					return true
				}
			}
		}
	}
	return true
}

func (esu *eventsubscriptionUtils) GetSystemIDFromURI(resourceURIArr []string, resourceOID string) string {
	systemID := resourceURIArr[4]
	if strings.Contains(resourceOID, "/TaskService/Tasks") {
		systemID = resourceURIArr[5]
	}
	if strings.Contains(resourceOID, "UpdateService/FirmwareInventory") {
		systemID = strings.Split(resourceURIArr[5], ".")[0]
	}
	return systemID
}

func (esu *eventsubscriptionUtils) GetEventSubscriptionRequest() ([]byte, error) {
	l.Log.Info("Creating eventsubscription request payload...")
	var (
		bmcObj               *infraiov1.Bmc
		volumeDetails        []string
		subscribedCollection = map[string]bool{}
	)
	err := esu.ValidateMessageIDs(esu.eventSubObj.Spec.MessageIds)
	if err != nil {
		return []byte{}, err
	}
	reqPayload := eventsubscriptionRequest{
		Name:                 esu.eventSubObj.Spec.Name,
		Destination:          esu.eventSubObj.Spec.Destination,
		EventTypes:           esu.eventSubObj.Spec.EventTypes,
		MessageIds:           esu.eventSubObj.Spec.MessageIds,
		ResourceTypes:        esu.eventSubObj.Spec.ResourceTypes,
		Context:              esu.eventSubObj.Spec.Context,
		EventFormatType:      esu.eventSubObj.Spec.EventFormatType,
		SubordinateResources: esu.eventSubObj.Spec.SubordinateResources,
		Protocol:             "Redfish",
		SubscriptionType:     "RedfishEvent",
		DeliveryRetryPolicy:  "RetryForever",
		OriginResources:      []map[string]string{},
	}

	for _, resource := range esu.eventSubObj.Spec.OriginResources {
		if collection, ok := MapCollectionToResourceURI[resource]; ok {
			if collection == "All" {
				addSubscriptionToAllCollections(subscribedCollection)
				break
			} else {
				subscribedCollection[collection] = true
			}
		} else {
			objectDetails := strings.Split(resource, "/")
			kind := objectDetails[0]
			objName := objectDetails[len(objectDetails)-1]
			if kind == "Volume" {
				volumeDetails = strings.Split(objName, ".")
				objName = volumeDetails[0]
			}
			bmcObj = esu.commonRec.GetBmcObject(esu.ctx, constants.MetadataName, objName, esu.namespace)
			switch kind {
			case "BiosSetting":
				if _, ok := subscribedCollection["/redfish/v1/Systems"]; !ok {
					if _, ok = subscribedCollection["/redfish/v1/Systems/"+bmcObj.Status.BmcSystemID]; !ok {
						resourceObj := esu.commonRec.GetBiosObject(esu.ctx, constants.MetadataName, objName, esu.namespace)
						if resourceObj != nil {
							subscribedCollection[resourceObj.ObjectMeta.Annotations["odata.id"]] = true

						} else {
							return []byte{}, fmt.Errorf("failed to create eventsubscription request: Could not find Bios object ")
						}
					}
				}
			case "Bmc":
				if _, ok := subscribedCollection["/redfish/v1/Systems"]; !ok {
					if bmcObj != nil {
						subscribedCollection[bmcObj.ObjectMeta.Annotations["odata.id"]] = true
						removeSubordinateResourceSubscriptions(subscribedCollection, bmcObj.Status.BmcSystemID)
					} else {
						return []byte{}, fmt.Errorf("failed to create eventsubscription request: Could not find Bmc object ")
					}

				}
			case "BootOrderSetting":
				if _, ok := subscribedCollection["/redfish/v1/Systems"]; !ok {
					if _, ok = subscribedCollection["/redfish/v1/Systems/"+bmcObj.Status.BmcSystemID]; !ok {
						resourceObj := esu.commonRec.GetBootObject(esu.ctx, constants.MetadataName, objName, esu.namespace)
						if resourceObj != nil {
							subscribedCollection[resourceObj.ObjectMeta.Annotations["odata.id"]] = true
						} else {
							return []byte{}, fmt.Errorf("failed to create eventsubscription request: Could not find Boot object ")
						}
					}
				}
			case "Firmware":
				if _, ok := subscribedCollection["/redfish/v1/Managers"]; !ok {
					resourceObj := esu.commonRec.GetFirmwareObject(esu.ctx, constants.MetadataName, objName, esu.namespace)
					if resourceObj != nil {
						inventories := resourceObj.ObjectMeta.Annotations["odata.id"]
						getOriginResourcesForFirmware(subscribedCollection, inventories)
					} else {
						return []byte{}, fmt.Errorf("failed to create eventsubscription request: Could not find Firmware object ")
					}
				}
			case "Volume":
				if _, ok := subscribedCollection["/redfish/v1/Systems"]; !ok {
					if _, ok = subscribedCollection["/redfish/v1/Systems/"+bmcObj.Status.BmcSystemID]; !ok {
						volumeDetails := strings.Split(objName, ".")
						resourceObj := esu.commonRec.GetVolumeObject(esu.ctx, volumeDetails[0], esu.namespace)
						if resourceObj != nil {
							subscribedCollection[resourceObj.ObjectMeta.Annotations["odata.id"]] = true
						} else {
							return []byte{}, fmt.Errorf("failed to create eventsubscription request: Could not find Volume object ")
						}
					}
				}
			case "allResources":
				addSubscriptionToAllCollectionForBmc(subscribedCollection, bmcObj.Status.BmcSystemID)
			}
		}
	}

	for key := range subscribedCollection {
		resourceLink := map[string]string{
			"@odata.id": key,
		}
		reqPayload.OriginResources = append(reqPayload.OriginResources, resourceLink)
	}
	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		l.LogWithFields(esu.ctx).Error("Failed to marshal eventsubscription request payload " + err.Error())
	}
	return reqBody, nil
}

func removeSubordinateResourceSubscriptions(subscribingCollection map[string]bool, bmcSystemID string) {
	if _, ok := subscribingCollection["/redfish/v1/Systems/"+bmcSystemID+"/Bios"]; ok {
		delete(subscribingCollection, "/redfish/v1/Systems/"+bmcSystemID+"/Bios")
	}
	if _, ok := subscribingCollection["/redfish/v1/Systems/"+bmcSystemID+"/BootOptions"]; ok {
		delete(subscribingCollection, "/redfish/v1/Systems/"+bmcSystemID+"/BootOptions")
	}
}

func (esu *eventsubscriptionUtils) DeleteEventsubscription() bool {
	deleteResp, err := esu.eventsubRestClient.Delete(fmt.Sprintf("/redfish/v1/EventService/Subscriptions/%s", esu.eventSubObj.Status.ID), fmt.Sprintf("Deleting event subscription with ID %s", esu.eventSubObj.Status.ID))
	if err != nil {
		l.LogWithFields(esu.ctx).Error(fmt.Sprintf("error while deleting event subscription with ID %s", esu.eventSubObj.Status.ID))
		return false
	}
	var resp map[string]interface{}
	var done bool
	if deleteResp.StatusCode == http.StatusAccepted {
		done, resp = esu.commonUtil.MoniteringTaskmon(deleteResp.Header, esu.ctx, common.DELETEEVENTSUBSCRIPTION, esu.eventSubObj.ObjectMeta.Name)
		if done {
			l.LogWithFields(esu.ctx).Info(fmt.Sprintf("Successfully deleted event subscription with ID %s", esu.eventSubObj.Status.ID))
			return true
		}
	}
	l.LogWithFields(esu.ctx).Info(fmt.Sprintf("Could not delete event subsscription with ID %s. Failed with error: %v. Please try again..", esu.eventSubObj.Status.ID, resp))
	return false
}

func (esu *eventsubscriptionUtils) UpdateEventSubscriptionLabels() {
	esu.eventSubObj.ObjectMeta.Labels = map[string]string{}
	esu.eventSubObj.ObjectMeta.Labels["eventSubscriptionID"] = esu.eventSubObj.Status.ID
}

func (esu *eventsubscriptionUtils) MapOriginResources(subscriptionDetails map[string]interface{}) ([]string, error) {
	var originResources []string
	var resource string
	if originResourcesLinks, ok := subscriptionDetails["OriginResources"].([]interface{}); ok {
		for _, originResource := range originResourcesLinks {
			resourceOID := originResource.(map[string]interface{})["@odata.id"].(string)

			if value, ok := MapResourceURIToCollection[resourceOID]; ok {
				resource = value
			} else {
				resourceURI := strings.Split(resourceOID, "/")
				systemID := esu.GetSystemIDFromURI(resourceURI, resourceOID)
				bmcObj := esu.commonRec.GetBmcObject(esu.ctx, constants.StatusBmcSystemID, systemID, esu.namespace)
				if bmcObj != nil {
					objectName := bmcObj.ObjectMeta.Name
					switch {
					case strings.Contains(resourceOID, "/Volumes/"):
						volumeID := resourceURI[len(resourceURI)-1]
						volObjs := esu.commonRec.GetAllVolumeObjects(esu.ctx, bmcObj.Spec.BmcDetails.Address, esu.namespace)
						if volObjs != nil {
							for _, volObj := range volObjs {
								if volObj.Status.VolumeID == volumeID {
									objectName = bmcObj.Spec.BmcDetails.Address + "/" + volObj.Status.VolumeName
									resource = "Volume/" + objectName
								}
							}
						}
					case strings.Contains(resourceOID, "/Bios"):
						resource = "BiosSetting/" + objectName
					case strings.Contains(resourceOID, "/BootOptions"):
						resource = "BootOrderSetting/" + objectName
					case strings.Contains(resourceOID, "UpdateService/FirmwareInventory"):

						resource = "Firmware/" + objectName
					case strings.Contains(resourceOID, "/Systems/"):
						resource = "Bmc/" + objectName
					}
				} else {
					return []string{}, fmt.Errorf("Bmc with systemID %s does not exist in system", systemID)
				}
			}
			if !slices.Contains(originResources, resource) {
				originResources = append(originResources, resource)
			}
		}
	}
	return originResources, nil
}

func (esu *eventsubscriptionUtils) GetMessageRegistryDetails(bmcObj *infraiov1.Bmc, registryName string) ([]string, []map[string]interface{}) {
	uri := "/redfish/v1/Registries/"
	var registryIDs []string
	var registryResponses []map[string]interface{}
	resp, sCode, _ := esu.eventsubRestClient.Get(uri, fmt.Sprintf("Fetching registries details for %s BMC", bmcObj.Spec.BmcDetails.Address))
	if sCode == http.StatusOK {
		for _, member := range resp["Members"].([]interface{}) {
			val := member.(map[string]interface{})["@odata.id"].(string)
			if strings.Contains(val, registryName) {
				res, statusCode, _ := esu.eventsubRestClient.Get(val, fmt.Sprintf("Fetching bios attribute registry details for %s BMC", bmcObj.Spec.BmcDetails.Address))
				if statusCode == http.StatusOK {
					for _, mem := range res["Location"].([]interface{}) {
						if mem.(map[string]interface{})["Language"].(string) == "en" {
							url := mem.(map[string]interface{})["Uri"].(string)
							r, s, _ := esu.eventsubRestClient.Get(url, fmt.Sprintf("Fetching bios attribute registry JSON for %s BMC", bmcObj.Spec.BmcDetails.Address))

							if s == http.StatusOK {
								registryIDs = append(registryIDs, r["RegistryPrefix"].(string)+"."+r["RegistryVersion"].(string))
								registryResponses = append(registryResponses, r)
							}
						}
					}
				}
			}
		}
	}
	return registryIDs, registryResponses
}

func (esu *eventsubscriptionUtils) ValidateMessageIDs(messageIDs []string) error {
	for _, messageID := range messageIDs {
		registryPrefix, registryVersion, messageKey := getMessageDetails(messageID)
		if strings.Contains(strings.ToLower(registryPrefix), "iloevents") {
			objName := strings.ToLower(registryPrefix) + "." + registryVersion
			messageRegistry := esu.commonRec.GetEventMessageRegistryObject(esu.ctx, constants.MetadataName, objName, esu.namespace)
			if messageRegistry != nil {
				if _, ok := messageRegistry.Spec.Messages[messageKey]; !ok {
					l.LogWithFields(esu.ctx).Errorf("message ID %s not found in the message registry %s. Please enter a valid message ID", messageKey, messageRegistry.Spec.Name)
					return fmt.Errorf("failed to create event subscription request")
				}
			} else {
				return fmt.Errorf("message registry not found for message ID %s", messageID)
			}
		}
	}
	return nil
}

func getMessageDetails(message string) (string, string, string) {
	message = strings.Replace(message, ".", "-", 1)
	index := strings.LastIndex(message, ".")
	parsedMessage := message[:index] + strings.Replace(message[index:], ".", "-", 1)
	messageDetails := strings.Split(parsedMessage, "-")
	return messageDetails[0], messageDetails[1], messageDetails[2]
}

func addSubscriptionToAllCollections(subscribingCollection map[string]bool) {
	for collectionURI := range MapResourceURIToCollection {
		subscribingCollection[collectionURI] = true
	}
}

func addSubscriptionToAllCollectionForBmc(subscribedCollection map[string]bool, bmcSystemID string) {
	subscribedCollection["/redfish/v1/Systems/"+bmcSystemID] = true
	subscribedCollection["/redfish/v1/Managers/"+bmcSystemID] = true
	subscribedCollection["/redfish/v1/Chassis/"+bmcSystemID] = true
}

func getOriginResourcesForFirmware(subscribedCollection map[string]bool, inventories string) {
	listOfInventories := strings.Split(inventories, ",")
	for _, item := range listOfInventories {
		subscribedCollection[item] = true
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventsubscriptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.Eventsubscription{}).
		WithEventFilter(utils.IgnoreStatusUpdate()).
		Complete(r)
}

// GetEventSubscriptionUtils will create and return an object of eventsubscriptionUtils
func GetEventSubscriptionUtils(ctx context.Context, restClient restclient.RestClientInterface, commonRec utils.ReconcilerInterface, eventSubObj *infraiov1.Eventsubscription, ns string) EventSubscriptionInterface {
	return &eventsubscriptionUtils{
		ctx:                ctx,
		eventsubRestClient: restClient,
		commonRec:          commonRec,
		eventSubObj:        eventSubObj,
		commonUtil:         common.GetCommonUtils(restClient),
		namespace:          ns,
	}
}
