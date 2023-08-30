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
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	eventsubscription "github.com/ODIM-Project/BMCOperator/controllers/eventsubscription"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
)

// AccommodateEventSubscriptionDetails will accomodate the newly created event subscriptions on ODIM into BMC operator
func (r PollingReconciler) AccommodateEventSubscriptionDetails(ctx context.Context, restClient restclient.RestClientInterface, defaultEventsubscriptionDestination string) {
	mapOfEventSubscriptions := make(map[string]bool) //mapOfEventSubscriptions contains a eventsubscriptionIDs of the eventsubscriptions present in the operator

	res, _, err := restClient.Get("/redfish/v1/EventService/Subscriptions", "Fetching all the event subscriptions present on ODIM")
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get event subscriptions")
	}

	eventsubscriptionObject := &infraiov1.Eventsubscription{}
	eventsubUtils := eventsubscription.GetEventSubscriptionUtils(ctx, r.pollRestClient, r.commonRec, eventsubscriptionObject, r.namespace)

	eventsubObjs := r.commonRec.GetAllEventSubscriptionObjects(ctx, r.namespace)
	createEventsubscriptionsMap(eventsubObjs, mapOfEventSubscriptions)

	if subscriptions, ok := res["Members"].([]interface{}); ok {
		for _, subscription := range subscriptions {
			reqURL := subscription.(map[string]interface{})["@odata.id"].(string)

			subscriptionDetails, statusCode, err := restClient.Get(reqURL, fmt.Sprintf("Fetching %s eventsubscription details..", reqURL[len(reqURL)-1:]))

			if err != nil {
				l.LogWithFields(ctx).Error(fmt.Sprintf("error while fetching %s eventsubscription details:", reqURL[len(reqURL)-1:]) + err.Error())
			} else if statusCode != http.StatusOK {
				l.LogWithFields(ctx).Info(fmt.Sprintf("Failed to get %s eventsubscription details, try again.", reqURL[len(reqURL)-1:]))
			} else {
				//scenario: If eventsubscription object is present on operator then check and update the status according to the object present on ODIM
				if subscriptionDetails["Destination"].(string) != defaultEventsubscriptionDestination {
					if _, ok := mapOfEventSubscriptions[subscriptionDetails["Id"].(string)]; ok {
						mapOfEventSubscriptions[subscriptionDetails["Id"].(string)] = true
						err := r.checkAndUpdateEventSubscriptionObject(ctx, subscriptionDetails, subscriptionDetails["Id"].(string), r.namespace, eventsubUtils)
						if err != nil {
							l.LogWithFields(ctx).Error(err)
						}
					} else {
						// scenario: If the object does not exist on operator then create one
						isCreated := r.CreateEventsubscriptionObject(ctx, subscriptionDetails, r.namespace, eventsubUtils)
						if !isCreated {
							l.LogWithFields(ctx).Error("Failed to create event subscription object")
						} else {
							mapOfEventSubscriptions[subscriptionDetails["Id"].(string)] = true
							l.LogWithFields(ctx).Infof("Successfully created eventsubscription object %s", subscriptionDetails["Name"].(string))
						}
					}
				}
			}
		}
	}
	r.RemoveSubscriptions(mapOfEventSubscriptions)
}

// RemoveSubscriptions deletes eventsubscription object which are not present on ODIM from operator
func (r PollingReconciler) RemoveSubscriptions(mapOfEventSubscriptions map[string]bool) {
	for eventsubID, isAdded := range mapOfEventSubscriptions {
		if !isAdded {
			existingEventsubObj := r.commonRec.GetEventsubscriptionObject(r.ctx, constants.StatusEventsubscriptionID, eventsubID, r.namespace)
			if existingEventsubObj != nil {
				controllerutil.RemoveFinalizer(existingEventsubObj, constants.EventsubscriptionFinalizer)
				err := r.Client.Update(r.ctx, existingEventsubObj)
				if err != nil {
					l.LogWithFields(r.ctx).Errorf("error while updating eventsubscription object for deletion with ID %s: %s", eventsubID, err.Error())
					return
				}
				err = r.Delete(r.ctx, existingEventsubObj)
				if err != nil {
					l.LogWithFields(r.ctx).Errorf("error while deleting eventsubscription object with ID %s: %s", eventsubID, err.Error())
					return
				}
			}
			delete(mapOfEventSubscriptions, eventsubID)
		}
	}

}

// RevertEventSubscriptionDetails will revert the event subscriptions on ODIM which does not exist in BMC operator
func (r PollingReconciler) RevertEventSubscriptionDetails(ctx context.Context, restClient restclient.RestClientInterface, defaultEventsubscriptionDestination string) {
	res, _, err := restClient.Get("/redfish/v1/EventService/Subscriptions", "Fetching all the event subscriptions present on ODIM")
	if err != nil {
		l.LogWithFields(ctx).Errorf("Failed to get event subscriptions")
	}
	mapOfEventSubscriptions := make(map[string]bool)
	eventsubObjs := r.commonRec.GetAllEventSubscriptionObjects(ctx, r.namespace)
	createEventsubscriptionsMap(eventsubObjs, mapOfEventSubscriptions)

	if subscriptions, ok := res["Members"].([]interface{}); ok {
		for _, subscription := range subscriptions {
			reqURL := subscription.(map[string]interface{})["@odata.id"].(string)

			subscriptionDetails, statusCode, err := restClient.Get(reqURL, fmt.Sprintf("Fetching %s eventsubscription details..", reqURL[len(reqURL)-1:]))

			if err != nil {
				l.LogWithFields(ctx).Error(fmt.Sprintf("error while fetching %s eventsubscription details:", reqURL[len(reqURL)-1:]) + err.Error())
			} else if statusCode != http.StatusOK {
				l.LogWithFields(ctx).Info(fmt.Sprintf("Failed to get %s eventsubscription details, try again.", reqURL[len(reqURL)-1:]))
			} else {
				if subscriptionDetails["Destination"].(string) != defaultEventsubscriptionDestination {
					subscriptionID := subscriptionDetails["Id"].(string)
					if _, ok := mapOfEventSubscriptions[subscriptionID]; ok {
						mapOfEventSubscriptions[subscriptionID] = true
					} else {
						isDeleted := r.DeleteEventsubscriptionFromODIM(ctx, subscriptionID, restClient)
						if !isDeleted {
							l.LogWithFields(ctx).Infof("Could not delete event subscription from ODIM with ID: %s", subscriptionID)
						}
					}
				}
			}
		}
	}
	if eventsubObjs != nil {
		for subscriptionID, isAdded := range mapOfEventSubscriptions {
			if !isAdded {
				r.AddEventSubscription(subscriptionID)
			}
		}
	}
}

// AddEventSubscription sends create request to ODIM for creating event subscription
func (r PollingReconciler) AddEventSubscription(subscriptionID string) {
	eventsubObj := r.commonRec.GetEventsubscriptionObject(r.ctx, constants.StatusEventsubscriptionID, subscriptionID, r.namespace)
	if eventsubObj != nil {
		eventsubUtils := eventsubscription.GetEventSubscriptionUtils(r.ctx, r.pollRestClient, r.commonRec, eventsubObj, r.namespace)
		req, err := eventsubUtils.GetEventSubscriptionRequest()
		if err != nil {
			l.LogWithFields(r.ctx).Errorf("could not create eventsubscription request for object %s: %s", eventsubObj.ObjectMeta.Name, err.Error())
			return
		}
		resp, err := r.pollRestClient.Post("/redfish/v1/EventService/Subscriptions", "Posting eventsubscription creation payload...", req)
		if err != nil {
			l.LogWithFields(r.ctx).Errorf("error while creating eventsubscription for object %s: %s", eventsubObj.ObjectMeta.Name, err.Error())
			return
		}
		if resp.StatusCode == http.StatusAccepted {
			done, _ := r.commonUtil.MoniteringTaskmon(resp.Header, r.ctx, common.EVENTSUBSCRIPTION, eventsubObj.ObjectMeta.Name)
			if done {
				l.LogWithFields(r.ctx).Info("\nEventsubscription created successfully!\n")
				l.LogWithFields(r.ctx).Info("Updating Eventsubscription status...")
				statusUpdated := eventsubUtils.UpdateEventsubscriptionStatus("")
				if !statusUpdated {
					l.LogWithFields(r.ctx).Info("Eventsubscription created successfully but failed to update the status")
				} else {
					l.LogWithFields(r.ctx).Info("Eventsubscription status updated successfully!")
				}
				return
			}
		}
		l.LogWithFields(r.ctx).Errorf("Could not create event subscription %s", eventsubObj.ObjectMeta.Name)
	}

}

func (r PollingReconciler) checkAndUpdateEventSubscriptionObject(ctx context.Context, eventsubscriptionResp map[string]interface{}, eventsubscriptionID, ns string, eventsubUtils eventsubscription.EventSubscriptionInterface) error {
	existingEventsubObj := r.commonRec.GetEventsubscriptionObject(ctx, constants.StatusEventsubscriptionID, eventsubscriptionID, ns)

	eventsubscriptionStatus := infraiov1.EventsubscriptionStatus{
		ID:               eventsubscriptionResp["Id"].(string),
		Destination:      eventsubscriptionResp["Destination"].(string),
		Context:          eventsubscriptionResp["Context"].(string),
		Protocol:         eventsubscriptionResp["Protocol"].(string),
		SubscriptionType: eventsubscriptionResp["SubscriptionType"].(string),
	}
	if name, ok := eventsubscriptionResp["Name"].(string); ok {
		eventsubscriptionStatus.Name = name
	}

	if eventTypes, ok := eventsubscriptionResp["EventTypes"].([]interface{}); ok {
		eventsubscriptionStatus.EventTypes = utils.ConvertInterfaceToStringArray(eventTypes)
	}
	if messageIDs, ok := eventsubscriptionResp["MessageIds"].([]interface{}); ok {
		eventsubscriptionStatus.MessageIds = utils.ConvertInterfaceToStringArray(messageIDs)
	}

	if resourceTypes, ok := eventsubscriptionResp["ResourceTypes"].([]interface{}); ok {
		eventsubscriptionStatus.ResourceTypes = utils.ConvertInterfaceToStringArray(resourceTypes)
	}
	originResources, err := eventsubUtils.MapOriginResources(eventsubscriptionResp)
	if err != nil {
		return fmt.Errorf("Failed to update eventsubscription object: %s", err.Error())
	}
	if len(originResources) > 0 {
		eventsubscriptionStatus.OriginResources = originResources
	}

	if existingEventsubObj != nil {
		if !reflect.DeepEqual(existingEventsubObj.Status, eventsubscriptionStatus) {
			existingEventsubObj.Status = eventsubscriptionStatus
			err := r.Client.Status().Update(ctx, existingEventsubObj)
			if err != nil {
				l.LogWithFields(ctx).Errorf("Error: Updating eventsubscription status for %s: %s", existingEventsubObj.ObjectMeta.Name, err.Error())
			}
		}
	}
	return nil
}

// CreateEventsubscriptionObject will create a new event subscription object in BMC operator
func (r PollingReconciler) CreateEventsubscriptionObject(ctx context.Context, eventsubscriptionResp map[string]interface{}, ns string, eventsubUtils eventsubscription.EventSubscriptionInterface) bool {
	originResources, err := eventsubUtils.MapOriginResources(eventsubscriptionResp)
	if err != nil {
		l.LogWithFields(r.ctx).Error(err.Error())
		return false
	}
	objectCreated := r.commonRec.CreateEventSubscriptionObject(ctx, eventsubscriptionResp, ns, originResources)
	return objectCreated
}

// DeleteEventsubscriptionFromODIM will delete the event subscription from ODIM
func (r PollingReconciler) DeleteEventsubscriptionFromODIM(ctx context.Context, subscriptionID string, restClient restclient.RestClientInterface) bool {
	deleteResp, err := restClient.Delete(fmt.Sprintf("/redfish/v1/EventService/Subscriptions/%s", subscriptionID), fmt.Sprintf("Deleting event subscription with ID %s", subscriptionID))
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("error while deleting event subscription with ID %s from ODIM", subscriptionID))
	}
	if deleteResp.StatusCode == http.StatusOK {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Successfully deleted event subscription with ID %s from ODIM", subscriptionID))
		return true
	}
	return false
}

func createEventsubscriptionsMap(eventsubObjs *[]infraiov1.Eventsubscription, mapOfEventSubscriptions map[string]bool) {
	if eventsubObjs != nil {
		for _, eventsubscription := range *eventsubObjs {
			if eventsubscription.Status.ID != "" {
				mapOfEventSubscriptions[eventsubscription.Status.ID] = false
			}
		}
	}
}
