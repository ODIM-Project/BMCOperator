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
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
)

func Test_createEventsubscriptionsMap(t *testing.T) {
	mapOfEventSubscriptions := make(map[string]bool)
	eventsubObjs := &[]infraiov1.Eventsubscription{
		infraiov1.Eventsubscription{
			Status: infraiov1.EventsubscriptionStatus{
				ID:               "123",
				Destination:      "https://10.24.1.14:8000/Destination",
				Name:             "EventSub1",
				Context:          "Events",
				Protocol:         "Redfish",
				SubscriptionType: "RedfishEvent",
			},
		},
		infraiov1.Eventsubscription{
			Status: infraiov1.EventsubscriptionStatus{
				ID:               "1234",
				Destination:      "https://10.24.1.14:8001/Destination",
				Name:             "EventSub2",
				Context:          "TestEvent",
				Protocol:         "Redfish",
				SubscriptionType: "RedfishEvent",
			},
		},
	}
	expectedMapOfEventSubscriptions := map[string]bool{
		"123":  false,
		"1234": false,
	}
	createEventsubscriptionsMap(eventsubObjs, mapOfEventSubscriptions)
	if !reflect.DeepEqual(mapOfEventSubscriptions, expectedMapOfEventSubscriptions) {
		t.Errorf("createEventsubscriptionsMap(): Expected :%v but got: %v", eventsubObjs, mapOfEventSubscriptions)
	}
}

func Test_CreateEventsubscriptionObject(t *testing.T) {
	mockUtil := mockEventSubscriptionUtil{name: "eventsubscription"}
	reconciler := getPollingReconcilerObject()

	eventsubResp := map[string]interface{}{
		"ID":               "1234",
		"Destination":      "https://10.24.1.14:8001/Destination",
		"Name":             "EventSub2",
		"Context":          "TestEvent",
		"Protocol":         "Redfish",
		"SubscriptionType": "RedfishEvent",
	}

	iscreated := reconciler.CreateEventsubscriptionObject(context.Background(), eventsubResp, "bmc-op", &mockUtil)
	if !iscreated {
		t.Errorf("CreateEventsubscriptionObject(): Expected true but got: %v", iscreated)
	}
}

func Test_DeleteEventsubscriptionFromODIM(t *testing.T) {
	reconciler := getPollingReconcilerObject()

	isDeleted := reconciler.DeleteEventsubscriptionFromODIM(context.Background(), "1234", reconciler.pollRestClient)
	if !isDeleted {
		t.Errorf("DeleteEventsubscriptionFromODIM(): Expected true but got: %v", isDeleted)
	}
}

func getPollingReconcilerObject() PollingReconciler {
	commonRecObj := MockCommonRec{variable: "eventSubscriptionObject"}

	reconciler := PollingReconciler{
		Client:         &MockClient{},
		Scheme:         nil,
		odimObj:        nil,
		bmcObject:      nil,
		commonRec:      &commonRecObj,
		pollRestClient: &mockRestClient{},
		ctx:            context.Background(),
	}
	return reconciler
}
