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

var commonRecObj = MockCommonRec{variable: "eventSubscriptionObject"}
var mockRC = mockRestClient{}

func Test_GetEventSubscriptionRequest(t *testing.T) {
	eventSubObj := &infraiov1.Eventsubscription{
		Spec: infraiov1.EventsubscriptionSpec{
			Destination:     "https://10.24.1.14:8093/Destination",
			Context:         "ODIMRA_Event",
			EventTypes:      []string{"Alert"},
			ResourceTypes:   []string{"ComputerSystem"},
			OriginResources: []string{"systemCollection"},
		},
	}
	eventSubUtils := &eventsubscriptionUtils{
		ctx:                context.TODO(),
		eventsubRestClient: &mockRC,
		commonRec:          &commonRecObj,
		eventSubObj:        eventSubObj,
		commonUtil:         nil,
		namespace:          "bmc-op",
	}

	expectedResp := "{\"Destination\":\"https://10.24.1.14:8093/Destination\",\"EventTypes\":[\"Alert\"],\"ResourceTypes\":[\"ComputerSystem\"],\"Context\":\"ODIMRA_Event\",\"SubordinateResources\":false,\"OriginResources\":[{\"@odata.id\":\"/redfish/v1/Systems\"}],\"Protocol\":\"Redfish\",\"SubscriptionType\":\"RedfishEvent\",\"DeliveryRetryPolicy\":\"RetryForever\"}"

	eventSubscriptionRequest, _ := eventSubUtils.GetEventSubscriptionRequest()

	if !reflect.DeepEqual(string(eventSubscriptionRequest), expectedResp) {
		t.Errorf("GetEventSubscriptionRequest(): expected: %s, recieved %s", expectedResp, string(eventSubscriptionRequest))
	}
}

func Test_DeleteEventsubscription(t *testing.T) {
	eventSubObj := &infraiov1.Eventsubscription{
		Spec: infraiov1.EventsubscriptionSpec{
			Destination:     "https://10.24.1.14:8093/Destination",
			Context:         "ODIMRA_Event",
			EventTypes:      []string{"Alert"},
			ResourceTypes:   []string{"ComputerSystem"},
			OriginResources: []string{"systemCollection"},
		},
		Status: infraiov1.EventsubscriptionStatus{
			ID:              "1234",
			Destination:     "https://10.24.1.14:8093/Destination",
			Context:         "ODIMRA_Event",
			EventTypes:      []string{"Alert"},
			ResourceTypes:   []string{"ComputerSystem"},
			OriginResources: []string{"systemCollection"},
		},
	}
	eventSubUtils := &eventsubscriptionUtils{
		ctx:                context.TODO(),
		eventsubRestClient: &mockRC,
		commonRec:          &commonRecObj,
		eventSubObj:        eventSubObj,
		commonUtil:         nil,
		namespace:          "bmc-op",
	}

	deleteResp := eventSubUtils.DeleteEventsubscription()
	if !deleteResp {
		t.Errorf("DeleteEventsubscription(): expected: %v, recieved %v", true, deleteResp)
	}
}
