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
	"flag"
	"net/http"
	"os"

	"github.com/ODIM-Project/BMCOperator/config/constants"
	sync "github.com/ODIM-Project/BMCOperator/controllers/pollData"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var podName = os.Getenv("POD_NAME")

// EventsClientReconciler helps to process the events receive from ODIM
type EventsClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var ecr *EventsClientReconciler

// Start initialize the event client listener
func Start(client client.Client, scheme *runtime.Scheme) {

	podName := os.Getenv("POD_NAME")
	ecr = &EventsClientReconciler{
		Client: client,
		Scheme: scheme,
	}

	transactionID := uuid.New().String()
	ctx := context.Background()
	ctx = l.CreateContextForLogging(ctx, transactionID, constants.BMCOPERATOR,
		constants.EventClientActionID, constants.EventClientActionName, podName)

	flag.Parse()
	app := newApp()

	apiServer, err := GetHTTPServerObj(ecr)
	if err != nil {
		l.LogWithFields(ctx).Fatal("service initialization failed: " + err.Error())
	}

	err = app.Run(iris.Server(apiServer))
	if err != nil {
		l.LogWithFields(ctx).Error("Error while starting the server", err.Error())
	}
}

// newApp creates new iris instance for REST API
func newApp() *iris.Application {
	app := iris.New()
	app.Post("/OdimEvents", triggerReconciler)

	return app
}

// triggerReconciler initiate event process and trigger the appropriate reconciler
func triggerReconciler(ctx iris.Context) {
	var data interface{}
	ctxt := ctx.Request().Context()
	transactionID := uuid.New().String()
	ctxt = l.CreateContextForLogging(ctxt, transactionID, constants.BMCOPERATOR,
		constants.EventClientActionID, constants.EventClientActionName, podName)
	err := ctx.ReadJSON(&data)
	if err != nil {
		l.LogWithFields(ctxt).Error("Error while trying to collect data from request: ", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	event, err := json.Marshal(data)
	if err != nil {
		l.LogWithFields(ctxt).Error("error while marshalling event from ODIM: " + err.Error())
		ctx.StatusCode(http.StatusInternalServerError)
	}

	l.LogWithFields(ctxt).Debug("Received an event from ODIM :- " + string(event))

	var messageData sync.OdimEventMessage
	err = json.Unmarshal(event, &messageData)
	if err != nil {
		l.LogWithFields(ctxt).Error("error while un-marshalling event from ODIM: " + err.Error())
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	for _, event := range messageData.Events {
		sync.ProcessOdimEvent(ctxt, ecr.Client, ecr.Scheme,
			messageData.Name, event)
	}
	ctx.StatusCode(http.StatusOK)
}
