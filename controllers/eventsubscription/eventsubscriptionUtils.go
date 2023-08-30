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

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	"k8s.io/apimachinery/pkg/types"
)

// EventSubscriptionInterface declares method signatures for EventSubscription
type EventSubscriptionInterface interface {
	CreateEventsubscription(eventSubObj *infraiov1.Eventsubscription, namespacedName types.NamespacedName) (bool, error)
	UpdateEventsubscriptionStatus(eventSubscriptionID string) bool
	GetEventSubscriptionRequest() ([]byte, error)
	DeleteEventsubscription() bool
	GetSystemIDFromURI(resourceURIArr []string, resourceOID string) string
	UpdateEventSubscriptionLabels()
	MapOriginResources(subscriptionDetails map[string]interface{}) ([]string, error)
	GetMessageRegistryDetails(bmcObj *infraiov1.Bmc, registryName string) ([]string, []map[string]interface{})
	ValidateMessageIDs(messageIDs []string) error
}

// MapCollectionToResourceURI maps collection to corresponding collection URI used while sending create request to ODIM
var MapCollectionToResourceURI = map[string]string{
	"systemCollection":  "/redfish/v1/Systems",
	"managerCollection": "/redfish/v1/Managers",
	"chassisCollection": "/redfish/v1/Chassis",
	"taskCollection":    "/redfish/v1/TaskService/Tasks",
	"fabricCollection":  "/redfish/v1/Fabrics",
	"allResources":      "All",
}

// MapResourceURIToCollection maps collection URI to collection name used to map the eventsubscription response from ODIM
var MapResourceURIToCollection = map[string]string{
	"/redfish/v1/Systems":           "systemCollection",
	"/redfish/v1/Managers":          "managerCollection",
	"/redfish/v1/Chassis":           "chassisCollection",
	"/redfish/v1/TaskService/Tasks": "taskCollection",
	"/redfish/v1/Fabrics":           "fabricCollection",
}

type eventsubscriptionUtils struct {
	ctx                context.Context
	eventsubRestClient restclient.RestClientInterface
	commonRec          utils.ReconcilerInterface
	commonUtil         common.CommonInterface
	eventSubObj        *infraiov1.Eventsubscription
	namespace          string
}

type eventsubscriptionRequest struct {
	Name                 string              `json:"Name,omitempty"`
	Destination          string              `json:"Destination"`
	EventTypes           []string            `json:"EventTypes,omitempty"`
	MessageIds           []string            `json:"MessageIds,omitempty"`
	ResourceTypes        []string            `json:"ResourceTypes,omitempty"`
	Context              string              `json:"Context"`
	EventFormatType      string              `json:"EventFormatType,omitempty"`
	SubordinateResources bool                `json:"SubordinateResources"`
	OriginResources      []map[string]string `json:"OriginResources,omitempty"`
	Protocol             string              `json:"Protocol"`
	SubscriptionType     string              `json:"SubscriptionType"`
	DeliveryRetryPolicy  string              `json:"DeliveryRetryPolicy"`
}
