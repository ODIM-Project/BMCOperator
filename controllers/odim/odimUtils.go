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

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
)

const (
	// DefaultEventSubscriptionName is the event subscription name for bmc operator event listener
	DefaultEventSubscriptionName = "BmcOperatorSubscription"
	// DefaultEventSubscriptionContext is the context of event subscription for bmc operator event listener
	DefaultEventSubscriptionContext = "Default Bmc-Operator EventSubscription"
)

type odimInterface interface {
	updateConnectionMethodVariantsStatus(connMethVarInfo map[string]string)
	checkOdimConnection() bool
	updateConnectionMethodVariants()
}

type odimUtils struct {
	ctx            context.Context
	odimRestClient restclient.RestClientInterface
	commonRec      utils.ReconcilerInterface
	odimObj        *infraiov1.Odim
}

// EventSubscriptionPayload is the payload of event subscription for server operator event listener
type EventSubscriptionPayload struct {
	Name                 string   `json:"Name"`
	Destination          string   `json:"Destination"`
	EventTypes           []string `json:"EventTypes"`
	MessageIds           []string `json:"MessageIds"`
	ResourceTypes        []string `json:"ResourceTypes"`
	Context              string   `json:"Context"`
	Protocol             string   `json:"Protocol"`
	SubscriptionType     string   `json:"SubscriptionType"`
	EventFormatType      string   `json:"EventFormatType"`
	SubordinateResources bool     `json:"SubordinateResources"`
	DeliveryRetryPolicy  string   `json:"DeliveryRetryPolicy"`
	OriginResources      []Link   `json:"OriginResources"`
}

// Link is struct defined for OriginResources in EventSubscriptionPayload
type Link struct {
	Oid string `json:"@odata.id"`
}

// GetEventSubscriptionPayload return the payload of event subscription for bmc operator event listener
func GetEventSubscriptionPayload(destination string) []byte {
	payload := EventSubscriptionPayload{
		Name:                 DefaultEventSubscriptionName,
		Destination:          destination + "/OdimEvents",
		EventTypes:           config.Data.EventSubscriptionEventTypes,
		MessageIds:           config.Data.EventSubscriptionMessageIds,
		ResourceTypes:        config.Data.EventSubscriptionResourceTypes,
		Context:              DefaultEventSubscriptionContext,
		Protocol:             "Redfish",
		SubscriptionType:     "RedfishEvent",
		EventFormatType:      "Event",
		SubordinateResources: true,
		DeliveryRetryPolicy:  "RetryForever",
		OriginResources: []Link{
			{
				Oid: "/redfish/v1/Systems",
			},
		},
	}

	body, _ := json.Marshal(payload)
	return body
}
