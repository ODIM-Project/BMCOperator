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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"

	Error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	eventsClient "github.com/ODIM-Project/BMCOperator/controllers/eventsClient"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/google/uuid"
)

// OdimReconciler reconciles a Odim object
type OdimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const odimFinalizer = "infra.io.odim/finalizer"

var podName = os.Getenv("POD_NAME")

//+kubebuilder:rbac:groups=infra.io.odimra,resources=odims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.io.odimra,resources=odims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.io.odimra,resources=odims/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Odim object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OdimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//commonReconciler object creation
	transactionId := uuid.New()
	ctx = l.CreateContextForLogging(ctx, transactionId.String(), constants.BmcOperator, constants.ODIMObjectOperationActionID, constants.ODIMObjectOperationActionName, podName)
	commonRec := utils.GetCommonReconciler(r.Client, r.Scheme)
	odimObj := &infraiov1.Odim{}
	// Fetch the Odim object
	err := r.Get(ctx, req.NamespacedName, odimObj)
	if err != nil {
		if Error.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	odimRestClient, err := restclient.NewRestClient(ctx, odimObj, commonRec.(*utils.CommonReconciler), constants.BMCOPERATOR)
	if err != nil {
		l.LogWithFields(ctx).Error("Failed to get rest client for ODIM" + err.Error())
		return ctrl.Result{}, err
	}
	//get odimUtil object
	odimUtil := getOdimUtils(ctx, odimRestClient, commonRec, odimObj)
	l.LogWithFields(ctx).Info("Checking odim presence")
	//Check redfish uri to see if connected
	//Checking for deletion
	isOdimMarkedToBeDeleted := odimObj.GetDeletionTimestamp() != nil
	if isOdimMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(odimObj, odimFinalizer) {
			isEventSubscriptionRemoved := removeEventsSubscription(ctx, odimRestClient)
			if isEventSubscriptionRemoved {
				controllerutil.RemoveFinalizer(odimObj, odimFinalizer)
				err = r.Update(ctx, odimObj)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else {
				l.LogWithFields(ctx).Error("cannot remove subscription from odim", err)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	connected := odimUtil.checkOdimConnection()
	if connected {
		l.LogWithFields(ctx).Info("Odim successfully registered!, updating connection methods")
		odimUtil.updateConnectionMethodVariants()
		l.LogWithFields(ctx).Info("Successfully added all connection methods!")
	} else {
		r.Delete(ctx, odimObj)
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(odimObj, odimFinalizer) {
		controllerutil.AddFinalizer(odimObj, odimFinalizer)
		err := r.Update(ctx, odimObj)
		if err != nil {
			l.LogWithFields(ctx).Error("Error while updating odim object after adding finalizer", err)
			return ctrl.Result{}, err
		}
	}
	commonRec.GetUpdatedOdimObject(ctx, req.NamespacedName, odimObj)
	// starting event listener
	go eventsClient.Start(r.Client, r.Scheme)
	// Creating default event subscription so that odim will send the events to the server operator listener
	err = CreateEventSubscription(ctx, odimRestClient, odimObj.Spec.EventListenerHost)
	if err != nil {
		l.LogWithFields(ctx).Error("error while creating the event subscription ", err.Error())
		return ctrl.Result{}, nil
	}
	l.LogWithFields(ctx).Debugf("event subscription is created with destination %s", odimObj.Spec.EventListenerHost)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OdimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infraiov1.Odim{}).
		WithEventFilter(utils.IgnoreStatusUpdate()).
		Complete(r)
}

// updateConnectionMethodVariantsStatus updates the connection method variants in status of the odim object
func (ou *odimUtils) updateConnectionMethodVariantsStatus(connMethVarInfo map[string]string) {
	ou.odimObj.Status.ConnMethVariants = connMethVarInfo //update printer column
	err := ou.commonRec.GetCommonReconcilerClient().Status().Update(ou.ctx, ou.odimObj)
	if err != nil {
		l.LogWithFields(ou.ctx).Error("Error: Updating plugin status of ODIM" + err.Error())
	}
}

// checkOdimConnection checks if we are able to connect odim
func (ou *odimUtils) checkOdimConnection() bool {
	resp, _, err := ou.odimRestClient.Get("/redfish/v1/", "Fetching response on service root..")
	if err != nil {
		l.LogWithFields(ou.ctx).Error("Not able to connect to ODIM", err.Error())
		ou.commonRec.UpdateOdimStatus(ou.ctx, "Not Connected", ou.odimObj)
		return false
	}
	if resp["@odata.id"] != nil {
		ou.commonRec.UpdateOdimStatus(ou.ctx, "Connected", ou.odimObj)
		return true
	} else {
		ou.commonRec.UpdateOdimStatus(ou.ctx, "Not Connected", ou.odimObj)
		return false
	}
}

// updateConnectionMethodVariants updates the connection method variants in the object
func (ou *odimUtils) updateConnectionMethodVariants() {
	connMethVarInfo := make(map[string]string)
	ConnMeths, _, err := ou.odimRestClient.Get("/redfish/v1/AggregationService/ConnectionMethods", "Fetching connection methods for ODIM..")
	if err != nil {
		l.LogWithFields(ou.ctx).Debug("Unable to get connection methods " + err.Error())
		return
	}
	if ConnMeths["Members"] != nil || len(ConnMeths["Members"].([]interface{})) != 0 {
		for _, memConnMethod := range ConnMeths["Members"].([]interface{}) {
			connMethInfo, _, err := ou.odimRestClient.Get(memConnMethod.(map[string]interface{})["@odata.id"].(string), "Fetching connection method details..")
			if err != nil {
				l.LogWithFields(ou.ctx).Error("Unable to get connection method info", err.Error())
				return
			}
			connMethVarInfo[connMethInfo["ConnectionMethodVariant"].(string)] = connMethInfo["@odata.id"].(string)
		}
	} else {
		l.LogWithFields(ou.ctx).Debug("No plugins present on ODIM")
		return
	}
	ou.updateConnectionMethodVariantsStatus(connMethVarInfo)
}

// getOdimUtils will return odimUtil struct with odim details
func getOdimUtils(ctx context.Context, restClient restclient.RestClientInterface, commonRec utils.ReconcilerInterface, odimObj *infraiov1.Odim) odimInterface {
	return &odimUtils{
		ctx:            ctx,
		odimRestClient: restClient,
		commonRec:      commonRec,
		odimObj:        odimObj,
	}
}

// CreateEventSubscription creates event subscription for server operator event listener
func CreateEventSubscription(ctx context.Context, restClient restclient.RestClientInterface, destination string) error {
	uri := "/redfish/v1/EventService/Subscriptions"
	body := GetEventSubscriptionPayload(destination)
	resp, err := restClient.Post(uri, "Creating default event subscription", body)
	if err != nil {
		return fmt.Errorf("error while adding default subscription for server operator: %s", err.Error())
	}
	if resp.StatusCode == http.StatusAccepted {
		commonUtil := common.GetCommonUtils(restClient)
		ok, taskResp := commonUtil.MoniteringTaskmon(resp.Header, ctx, common.EVENTSUBSCRIPTION, destination)
		if !ok {
			return fmt.Errorf("failed to create event subscription: task response %v", taskResp)
		}
	}
	return nil
}

func removeEventsSubscription(ctx context.Context, restClient restclient.RestClientInterface) bool {
	subscriptionsIds := []string{}
	subscriptionsCollectionUri := "/redfish/v1/EventService/Subscriptions"
	subscriptionsUrl := "/redfish/v1/EventService/Subscriptions/%s"
	resp, statusCode, err := restClient.Get(subscriptionsCollectionUri, "Getting all  event subscription collections")
	if err != nil {
		l.LogWithFields(ctx).Errorf("error while getting event subscription collections for bmc operator: %s", err.Error())
		return false
	}
	if statusCode == http.StatusOK {
		for _, mem := range resp["Members"].([]interface{}) {
			members := mem.(map[string]interface{})
			member, err := url.Parse(members["@odata.id"].(string))
			if err != nil {
				l.LogWithFields(ctx).Errorf("error while parsing the url %v", err)
				return false
			}
			subscriptionsIds = append(subscriptionsIds, path.Base(member.String()))
		}
	}
	// Checking for the operator subscription
	for _, subs := range subscriptionsIds {
		subUri := fmt.Sprintf(subscriptionsUrl, subs)
		resp, statusCode, err := restClient.Get(subUri, "Creating default event subscription")
		if err != nil {
			l.LogWithFields(ctx).Errorf("error while getting subscription for bmc operator: %s", err.Error())
			return false
		}
		if statusCode == http.StatusOK {
			if resp["Name"] == "BmcOperatorSubscription" {
				resp, err := restClient.Delete(subUri, "Deleting the operator events subscription")
				if err != nil {
					l.LogWithFields(ctx).Errorf("error while deleting subscription for bmc operator: %s", err.Error())
					return false
				}
				if resp.StatusCode == http.StatusAccepted {
					commonUtil := common.GetCommonUtils(restClient)
					done, _ := commonUtil.MoniteringTaskmon(resp.Header, ctx, common.DELETEEVENTSUBSCRIPTION, "BmcOperatorSubscription")
					if done {
						l.LogWithFields(ctx).Infof("deleted events subscription with ID %s", subs)
						return true
					}
				}
			}
		}
	}
	l.LogWithFields(ctx).Error("Received unexpected status code while deleting the events subscription")
	return false
}
