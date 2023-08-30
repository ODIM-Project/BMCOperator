// (C) Copyright [2022] Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.
package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	l "github.com/ODIM-Project/BMCOperator/logs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	updateFunc = func(e event.UpdateEvent) bool {
		return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
	}
)

type CommonReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

type ReconcilerInterface interface {
	//get objects
	GetBmcObject(ctx context.Context, field, value, ns string) *infraiov1.Bmc
	GetAllBmcObject(ctx context.Context, ns string) *[]infraiov1.Bmc
	GetBiosSchemaObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSchemaRegistry
	GetBiosObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSetting
	GetBootObject(ctx context.Context, field, value, ns string) *infraiov1.BootOrderSetting
	GetOdimObject(ctx context.Context, field, value, ns string) *infraiov1.Odim
	GetVolumeObject(ctx context.Context, bmcIP, ns string) *infraiov1.Volume
	GetAllVolumeObjects(ctx context.Context, bmcIP, ns string) []*infraiov1.Volume
	GetVolumeObjectByVolumeID(ctx context.Context, volumeID, ns string) *infraiov1.Volume
	GetEventsubscriptionObject(ctx context.Context, field, value, ns string) *infraiov1.Eventsubscription
	GetAllEventSubscriptionObjects(ctx context.Context, ns string) *[]infraiov1.Eventsubscription
	GetAllBiosSchemaRegistryObjects(ctx context.Context, ns string) *[]infraiov1.BiosSchemaRegistry
	//get attributes of objects
	GetAllVolumeObjectIds(ctx context.Context, bmc *infraiov1.Bmc, ns string) map[string][]string
	GetFirmwareObject(ctx context.Context, field, value, ns string) *infraiov1.Firmware
	GetEventMessageRegistryObject(ctx context.Context, field, value, ns string) *infraiov1.EventsMessageRegistry
	//create objects
	CreateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, bmcObj *infraiov1.Bmc) bool
	CreateBootOrderSettingObject(ctx context.Context, bootAttributes *infraiov1.BootSetting, bmcObj *infraiov1.Bmc) bool
	CheckAndCreateBiosSchemaObject(ctx context.Context, attributeResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool
	CreateEventSubscriptionObject(ctx context.Context, subscriptionDetails map[string]interface{}, ns string, originResources []string) bool
	CheckAndCreateEventMessageObject(ctx context.Context, messageRegistryResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool
	//update objects
	UpdateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, bmcObj *infraiov1.BiosSetting) bool
	UpdateBmcStatus(ctx context.Context, bmcObj *infraiov1.Bmc)
	UpdateOdimStatus(ctx context.Context, status string, odimObj *infraiov1.Odim)
	UpdateBmcObjectOnReset(ctx context.Context, bmcObject *infraiov1.Bmc, status string)
	UpdateVolumeStatus(ctx context.Context, volObject *infraiov1.Volume, volumeID, volumeName, capBytes, durableName, durableNameFormat string)
	UpdateEventsubscriptionStatus(ctx context.Context, eventsubObj *infraiov1.Eventsubscription, eventsubscriptionDetails map[string]interface{}, originResouces []string)
	//get updated objects
	GetUpdatedBmcObject(ctx context.Context, ns types.NamespacedName, bmcObj *infraiov1.Bmc)
	GetUpdatedOdimObject(ctx context.Context, ns types.NamespacedName, odimObj *infraiov1.Odim)
	GetUpdatedVolumeObject(ctx context.Context, ns types.NamespacedName, volObj *infraiov1.Volume)
	//delete objects
	DeleteBmcObject(ctx context.Context, bmcObj *infraiov1.Bmc)
	DeleteVolumeObject(ctx context.Context, volObj *infraiov1.Volume)
	GetUpdatedFirmwareObject(ctx context.Context, ns types.NamespacedName, firmObj *infraiov1.Firmware)
	GetUpdatedEventsubscriptionObjects(ctx context.Context, ns types.NamespacedName, eventSubObj *infraiov1.Eventsubscription)
	//common reconciler funcs
	GetCommonReconcilerClient() client.Client
	GetCommonReconcilerScheme() *runtime.Scheme
}

// GetCommonReconciler will return common Reconciler object
func GetCommonReconciler(c client.Client, s *runtime.Scheme) ReconcilerInterface {
	return &CommonReconciler{Client: c, Scheme: s}
}
func (c *CommonReconciler) GetCommonReconcilerClient() client.Client {
	return c.Client
}
func (c *CommonReconciler) GetCommonReconcilerScheme() *runtime.Scheme {
	return c.Scheme
}

// ignoreStatusUpdate ignores reconcile when status/metadata is updated
func IgnoreStatusUpdate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: updateFunc,
	}
}

// ----------------------------GET OBJECTS---------------------------------
// GetBmcObject is used to get bmc object details based on given field and value
func (r *CommonReconciler) GetBmcObject(ctx context.Context, field, value, ns string) *infraiov1.Bmc {
	list := &infraiov1.BmcList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any BMC object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, (fmt.Sprintf("Error fetching the BMC object for given field %s", field)))
		return nil
	}
	return &list.Items[0]
}

// GetAllBmcObject is used to get all bmc object details based on given namespace
func (r *CommonReconciler) GetAllBmcObject(ctx context.Context, ns string) *[]infraiov1.Bmc {
	list := &infraiov1.BmcList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error("Couldn't find any BMC object")
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, "Error fetching the BMC object details")
		return nil
	}
	return &list.Items
}

// GetBiosSchemaObject is used to get bios schema object details based on given field and value
func (r *CommonReconciler) GetBiosSchemaObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSchemaRegistry {
	list := &infraiov1.BiosSchemaRegistryList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any BIOS-SCHEMA object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching the BIOS-SCHEMA object for given field %s: %s", field, err.Error()))
		return nil
	}
	return &list.Items[0]
}

// GetBiosObject is used to get bios object details based on given field and value
func (r *CommonReconciler) GetBiosObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSetting {
	list := &infraiov1.BiosSettingList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Couldn't find any BIOS object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching the BIOS object for given field %s: %s", field, err.Error()))
		return nil
	}
	return &list.Items[0]
}

// GetBootObject is used to get bios object details based on given field and value
func (r *CommonReconciler) GetBootObject(ctx context.Context, field, value, ns string) *infraiov1.BootOrderSetting {
	list := &infraiov1.BootOrderSettingList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Couldn't find any BOOT object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching the BOOT object for given field %s: %s", field, err.Error()))
		return nil
	}
	return &list.Items[0]
}

func (r *CommonReconciler) GetOdimObject(ctx context.Context, field, value, ns string) *infraiov1.Odim {
	list := &infraiov1.OdimList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Couldn't find any ODIM object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching the ODIM object for given field %s: %s", field, err.Error()))
		return nil
	}
	return &list.Items[0]
}

func (r *CommonReconciler) GetVolumeObject(ctx context.Context, bmcIP, ns string) *infraiov1.Volume {
	list := &infraiov1.VolumeList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any Volume object for bmc: %s", bmcIP))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, (fmt.Sprintf("Error fetching the Volume object for bmc: %s", bmcIP)))
		return nil
	}
	for _, volObj := range list.Items {
		if strings.Contains(volObj.ObjectMeta.Name, bmcIP) {
			return &volObj
		}
	}
	return &list.Items[0]
}

func (r *CommonReconciler) GetAllVolumeObjects(ctx context.Context, bmcIP, ns string) []*infraiov1.Volume {
	list := &infraiov1.VolumeList{}
	volList := []*infraiov1.Volume{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any Volume object for bmc: %s", bmcIP))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, (fmt.Sprintf("Error fetching the Volume object for bmc: %s", bmcIP)))
		return nil
	}
	for _, volObj := range list.Items {
		if strings.Contains(volObj.ObjectMeta.Name, bmcIP) {
			volList = append(volList, &volObj)
		}
	}
	return volList
}

// GetVolumeObjectByVolumeID will fetch and return volume object based on volumeID
func (r *CommonReconciler) GetVolumeObjectByVolumeID(ctx context.Context, volumeID, ns string) *infraiov1.Volume {
	list := &infraiov1.VolumeList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any Volume object by volumeName %s:", volumeID), err.Error())
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, (fmt.Sprintf("Error fetching the Volume object %s: ", volumeID)), err.Error())
		return nil
	}
	for _, volObj := range list.Items {
		if volObj.Status.VolumeID == volumeID {
			return &volObj
		}
	}
	return &list.Items[0]
}

// -----------------------------------------GET OBJECT'S ATTRIBUTES----------------------------------------------

// GetAllVolumeObjectIds will get all the volume ids of specific bmc volume objects
func (r *CommonReconciler) GetAllVolumeObjectIds(ctx context.Context, bmc *infraiov1.Bmc, ns string) map[string][]string {
	list := &infraiov1.VolumeList{}
	volumeIds := map[string][]string{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any Volume objects in %s namespace", ns))
		return map[string][]string{}
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, (fmt.Sprintf("Error fetching Volume objects in %s namespace", ns)))
		return nil
	}
	for _, volume := range list.Items {
		if strings.Contains(volume.ObjectMeta.Name, bmc.ObjectMeta.Name) {
			if volume.Status.StorageControllerID == "" {
				continue
			}
			if volumeIds[volume.Status.StorageControllerID] == nil {
				var volIds []string
				volIds = append(volIds, volume.Status.VolumeID)
				volumeIds[volume.Status.StorageControllerID] = volIds
			} else {
				volIds := volumeIds[volume.Status.StorageControllerID]
				volIds = append(volIds, volume.Status.VolumeID)
				volumeIds[volume.Status.StorageControllerID] = volIds
			}
		}
	}
	return volumeIds
}

// GetFirmwareObject will fetch and return a firmware object based on the field and value
func (r *CommonReconciler) GetFirmwareObject(ctx context.Context, field, value, ns string) *infraiov1.Firmware {
	list := &infraiov1.FirmwareList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Info(fmt.Sprintf("Couldn't find any Firmware object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching the Firmware object for given field %s: %s", field, err.Error()))
		return nil
	}
	return &list.Items[0]
}

// GetEventsubscriptionObject will fetch the event subscription present in BMC operator based on the field and value and return the first object
func (r *CommonReconciler) GetEventsubscriptionObject(ctx context.Context, field, value, ns string) *infraiov1.Eventsubscription {
	list := &infraiov1.EventsubscriptionList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any eventsubscription object for value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, (fmt.Sprintf("Error fetching the BMC object for given field %s", field)))
		return nil
	}
	return &list.Items[0]
}

// GetAllEventSubscriptionObjects will return all the event subscription objects present in BMC operator under the requested namespace
func (r *CommonReconciler) GetAllEventSubscriptionObjects(ctx context.Context, ns string) *[]infraiov1.Eventsubscription {
	list := &infraiov1.EventsubscriptionList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error("Couldn't find any Event subscription object")
		return nil
	}

	if err != nil {
		l.LogWithFields(ctx).Error(err, "Error fetching the Event subscription details")
		return nil
	}
	return &list.Items
}

// GetAllBiosSchemaRegistryObjects will get all the bios schema registry objects
func (r *CommonReconciler) GetAllBiosSchemaRegistryObjects(ctx context.Context, ns string) *[]infraiov1.BiosSchemaRegistry {
	list := &infraiov1.BiosSchemaRegistryList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error("Couldn't find any Bios Schema Registry object")
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(err, "Error fetching Bios Schema Registry object")
		return nil
	}
	return &list.Items
}

// GetEventMessageRegistryObject fetch eventsmessageregistry object based on fields and values passed
func (r *CommonReconciler) GetEventMessageRegistryObject(ctx context.Context, field, value, ns string) *infraiov1.EventsMessageRegistry {
	list := &infraiov1.EventsMessageRegistryList{}
	opts := []client.ListOption{
		client.InNamespace(ns),
		client.MatchingFields{field: value},
	}
	err := r.Client.List(ctx, list, opts...)
	if len(list.Items) == 0 {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Couldn't find any event registry object value %s and field %s", value, field))
		return nil
	}
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching the event registry object value %s and field %s : %s", value, field, err.Error()))
		return nil
	}
	return &list.Items[0]
}

// -----------------------------------------CREATE OBJECTS----------------------------------------------

// createBiosSettingObject is used for creating bios setting object with same name and namespace as of bmc
func (r *CommonReconciler) CreateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, bmcObj *infraiov1.Bmc) bool {
	bios := infraiov1.BiosSetting{}
	bios.ObjectMeta.Name = bmcObj.ObjectMeta.Name
	bios.ObjectMeta.Namespace = bmcObj.ObjectMeta.Namespace
	bios.Spec.Bios = map[string]string{}
	bios.ObjectMeta.Annotations = map[string]string{}
	bios.ObjectMeta.Annotations["odata.id"] = "/redfish/v1/Systems/" + bmcObj.Status.BmcSystemID + "/Bios"
	err := r.Client.Create(context.Background(), &bios)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error while creating biosSetting object for %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
		return false
	}
	bios.Status = infraiov1.BiosSettingStatus{
		BiosAttributes: biosAttributes,
	}
	err = r.Client.Status().Update(context.Background(), &bios)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating status with bios attributes for %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
	}
	return true
}

// CreateBootOrderSettingObject is used for creating boot order object with same name and namespace as of bmc
func (r *CommonReconciler) CreateBootOrderSettingObject(ctx context.Context, bootAttributes *infraiov1.BootSetting, bmcObj *infraiov1.Bmc) bool {
	boot := infraiov1.BootOrderSetting{}
	boot.ObjectMeta.Name = bmcObj.ObjectMeta.Name
	boot.ObjectMeta.Namespace = bmcObj.ObjectMeta.Namespace
	boot.ObjectMeta.Annotations = map[string]string{}
	boot.ObjectMeta.Annotations["odata.id"] = "/redfish/v1/Systems/" + bmcObj.Status.BmcSystemID + "/BootOptions"
	err := r.Client.Create(context.Background(), &boot)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error while creating bootOrderSetting object for %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
		return false
	}
	boot.Status = infraiov1.BootOrderSettingsStatus{
		Boot: *bootAttributes,
	}
	err = r.Client.Status().Update(context.Background(), &boot)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating status with boot attributes for %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
	}
	return true
}

// CheckAndCreateBiosSchemaObject verifies if bios schema object is available else create it
func (r *CommonReconciler) CheckAndCreateBiosSchemaObject(ctx context.Context, attributeResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool {
	biosID := RemoveSpecialChar(attributeResp["Id"].(string))
	var attribute = make([]map[string]string, 0)
	attributes := attributeResp["RegistryEntries"].(map[string]interface{})["Attributes"]
	for _, val := range attributes.([]interface{}) {
		var attributeDetails = make(map[string]string)
		for k, v := range val.(map[string]interface{}) {
			if k == "Value" {
				res, _ := json.Marshal(v)
				attributeDetails[k] = string(res)
			} else {
				s := fmt.Sprintf("%v", v)
				attributeDetails[k] = s
			}

		}
		attribute = append(attribute, attributeDetails)
	}
	biosSchema := infraiov1.BiosSchemaRegistry{}
	biosSchema.ObjectMeta.Name = biosID
	biosSchema.ObjectMeta.Namespace = bmcObj.Namespace
	biosSchema.Spec.Name = attributeResp["Name"].(string)
	biosSchema.Spec.ID = attributeResp["Id"].(string)
	biosSchema.Spec.OwningEntity = attributeResp["OwningEntity"].(string)
	biosSchema.Spec.Attributes = attribute
	var supportedSystems []infraiov1.SupportedSystems
	supportedSystemsFromResp := attributeResp["SupportedSystems"].([]interface{})
	for _, system := range supportedSystemsFromResp {
		sys := system.(map[string]interface{})
		supportedSystem := infraiov1.SupportedSystems{}
		if sys["ProductName"] != nil {
			supportedSystem.ProductName = sys["ProductName"].(string)
		}
		if sys["SystemID"] != nil {
			supportedSystem.SystemID = sys["SystemID"].(string)
		}
		if sys["FirmwareVersion"] != nil {
			supportedSystem.FirmwareVersion = sys["FirmwareVersion"].(string)
		}
		supportedSystems = append(supportedSystems, supportedSystem)
	}
	biosSchema.Spec.SupportedSystems = supportedSystems
	key := client.ObjectKey{Namespace: bmcObj.Namespace, Name: biosID}
	err := r.Client.Get(ctx, key, &biosSchema)
	if err != nil {
		err := r.Client.Create(ctx, &biosSchema)
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf("Error while creating BIOS-SCHEMA object for %s BMC!: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
			return false
		}
		l.LogWithFields(ctx).Info(fmt.Sprintf("BIOS-SCHEMA object for %s BMC created with name: %s", bmcObj.Spec.BmcDetails.Address, biosID))
	}
	return true
}

// CreateEventSubscriptionObject creates an eventsubscription object in system
func (r *CommonReconciler) CreateEventSubscriptionObject(ctx context.Context, subscriptionDetails map[string]interface{}, ns string, originResources []string) bool {
	eventsubscriptionObj := infraiov1.Eventsubscription{}
	subscriptionName := r.GetEventSubscriptionObjectName(ctx, subscriptionDetails["Destination"].(string), subscriptionDetails["Name"].(string), ns)
	eventsubscriptionObj.ObjectMeta.Name = subscriptionName
	eventsubscriptionObj.ObjectMeta.Namespace = ns
	err := r.Client.Create(context.Background(), &eventsubscriptionObj)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error while creating eventsubscription object %s: %s", eventsubscriptionObj.ObjectMeta.Name, err.Error()))
		return false
	}

	eventsubscriptionObj.Status.ID = subscriptionDetails["Id"].(string)
	eventsubscriptionObj.Status.Destination = subscriptionDetails["Destination"].(string)
	eventsubscriptionObj.Status.Context = subscriptionDetails["Context"].(string)
	eventsubscriptionObj.Status.Protocol = subscriptionDetails["Protocol"].(string)
	eventsubscriptionObj.Status.SubscriptionType = subscriptionDetails["SubscriptionType"].(string)
	eventsubscriptionObj.Status.Name = subscriptionName

	if eventTypes, ok := subscriptionDetails["EventTypes"].([]interface{}); ok {
		eventsubscriptionObj.Status.EventTypes = ConvertInterfaceToStringArray(eventTypes)
	}
	if messageIDs, ok := subscriptionDetails["MessageIds"].([]interface{}); ok {
		eventsubscriptionObj.Status.MessageIds = ConvertInterfaceToStringArray(messageIDs)
	}

	if resourceTypes, ok := subscriptionDetails["ResourceTypes"].([]interface{}); ok {
		eventsubscriptionObj.Status.ResourceTypes = ConvertInterfaceToStringArray(resourceTypes)
	}
	if len(originResources) > 0 {
		eventsubscriptionObj.Status.OriginResources = originResources
	}
	err = r.Client.Status().Update(context.Background(), &eventsubscriptionObj)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating eventsubscription object status %s BMC: %s", eventsubscriptionObj.ObjectMeta.Name, err.Error()))
	}

	return true
}

// CheckAndCreateEventMessageObject checks if the message registry object already exist, if not then the object is created
func (r *CommonReconciler) CheckAndCreateEventMessageObject(ctx context.Context, messageRegistryResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool {
	messages := map[string]infraiov1.EventMessage{}
	registryID := RemoveSpecialChar(messageRegistryResp["Id"].(string))
	messageEntries := messageRegistryResp["Messages"].(map[string]interface{})
	for key, value := range messageEntries {
		messageDetails := value.(map[string]interface{})
		eventMessage := infraiov1.EventMessage{
			Description:  messageDetails["Description"].(string),
			Message:      messageDetails["Message"].(string),
			NumberOfArgs: fmt.Sprintf("%f", messageDetails["NumberOfArgs"].(float64)),
			Resolution:   messageDetails["Resolution"].(string),
			Severity:     messageDetails["Severity"].(string),
			ParamTypes:   ConvertInterfaceToStringArray(messageDetails["ParamTypes"].([]interface{})),
			Oem:          map[string]infraiov1.Oem{},
		}
		var dataType, healthCatagory, eventType string
		if oemDetails, ok := messageDetails["Oem"].(map[string]interface{}); ok {
			for oem, oemData := range oemDetails {
				oemProperties := oemData.(map[string]interface{})
				eventMessage.Oem[oem] = infraiov1.Oem{}
				if value, ok := oemProperties["@odata.type"]; ok {
					dataType = value.(string)
				}
				if value, ok := oemProperties["HealthCategory"]; ok {
					healthCatagory = value.(string)
				}
				if value, ok := oemProperties["Type"]; ok {
					eventType = value.(string)
				}
				eventMessage.Oem[oem] = infraiov1.Oem{
					OdataType:      dataType,
					HealthCategory: healthCatagory,
					Type:           eventType,
				}
			}
		}
		messages[key] = eventMessage
	}

	objName := strings.ToLower(messageRegistryResp["RegistryPrefix"].(string)) + "." + messageRegistryResp["RegistryVersion"].(string)
	messageRegistryObj := infraiov1.EventsMessageRegistry{}
	messageRegistryObj.ObjectMeta.Name = objName
	messageRegistryObj.ObjectMeta.Namespace = bmcObj.Namespace
	messageRegistryObj.Spec.Name = messageRegistryResp["Name"].(string)
	messageRegistryObj.Spec.ID = messageRegistryResp["Id"].(string)
	messageRegistryObj.Spec.OwningEntity = messageRegistryResp["OwningEntity"].(string)
	messageRegistryObj.Spec.RegistryPrefix = messageRegistryResp["RegistryPrefix"].(string)
	messageRegistryObj.Spec.RegistryVersion = messageRegistryResp["RegistryVersion"].(string)
	messageRegistryObj.Spec.Messages = messages

	key := client.ObjectKey{Namespace: bmcObj.Namespace, Name: objName}
	err := r.Client.Get(ctx, key, &messageRegistryObj)
	if err != nil {
		err := r.Client.Create(ctx, &messageRegistryObj)
		if err != nil {
			l.LogWithFields(ctx).Error(fmt.Sprintf("Error while creating EventMessageRegistry object for %s BMC!: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
			return false
		}
		l.LogWithFields(ctx).Info(fmt.Sprintf("EventMessageRegistry object for %s BMC created with name: %s", bmcObj.Spec.BmcDetails.Address, registryID))
	}
	return true
}

// ----------------------------------------UPDATE OBJECT STATUS---------------------------------------
// UpdateBiosSettingObject used to update the bios object
func (r *CommonReconciler) UpdateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, biosObj *infraiov1.BiosSetting) bool {
	biosObj.Spec.Bios = map[string]string{}
	biosObj.Status = infraiov1.BiosSettingStatus{
		BiosAttributes: biosAttributes,
	}
	err := r.Client.Status().Update(ctx, biosObj)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error: Updating status with bios attributes for %s BMC: %s", biosObj.Name, err.Error()))
	}
	return true
}

// UpdateBmcStatus used to update the bmc object
func (r *CommonReconciler) UpdateBmcStatus(ctx context.Context, bmcObj *infraiov1.Bmc) {
	err := r.Client.Status().Update(ctx, bmcObj)
	if err != nil {
		l.Log.Error(fmt.Sprintf("Error: Updating status of %s BMC: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
	}
	time.Sleep(time.Duration(40) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
}

// UpdateOdimStatus updates the status of odim object under status field
func (r *CommonReconciler) UpdateOdimStatus(ctx context.Context, status string, odimObj *infraiov1.Odim) {
	odimObj.Status.Status = status //update printer column
	err := r.Client.Status().Update(ctx, odimObj)
	if err != nil {
		l.LogWithFields(ctx).Error("Error: Updating status of ODIM" + err.Error())
	}
	time.Sleep(time.Duration(10) * time.Second) // NOTE: Do not delete this, this helps in proper update of the object
}

// UpdateBmcObjectOnReset updates the systemReset field in bmc object
func (r *CommonReconciler) UpdateBmcObjectOnReset(ctx context.Context, bmcObject *infraiov1.Bmc, status string) {
	bmcObject.Status.SystemReset = status
	err := r.Client.Status().Update(ctx, bmcObject)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Error: Updating Status of %s BMC after Reset: %s", bmcObject.Spec.BmcDetails.Address, err.Error())
	}
}

// UpdateVolumeStatus updates the volume object status
func (r *CommonReconciler) UpdateVolumeStatus(ctx context.Context, volObject *infraiov1.Volume, volumeID, volumeName, capBytes, durableName, durableNameFormat string) {
	volObject.Status.VolumeID = volumeID
	volObject.Status.VolumeName = volumeName
	volObject.Status.RAIDType = volObject.Spec.RAIDType
	volObject.Status.StorageControllerID = volObject.Spec.StorageControllerID
	volObject.Status.CapacityBytes = capBytes
	volObject.Status.Drives = volObject.Spec.Drives
	volObject.Status.Identifiers.DurableName = durableName
	volObject.Status.Identifiers.DurableNameFormat = durableNameFormat
	err := r.Client.Status().Update(ctx, volObject)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Error: Updating Status of %s Volume : %s", volObject.Status.VolumeName, err.Error())
	}
	time.Sleep(time.Duration(30) * time.Second)
}

// UpdateEventsubscriptionStatus will update the status of event subscription object
func (r *CommonReconciler) UpdateEventsubscriptionStatus(ctx context.Context, eventsubObj *infraiov1.Eventsubscription, eventsubscriptionDetails map[string]interface{}, originResources []string) {
	eventsubObj.Status.ID = eventsubscriptionDetails["Id"].(string)
	eventsubObj.Status.Destination = eventsubscriptionDetails["Destination"].(string)
	eventsubObj.Status.Protocol = eventsubscriptionDetails["Protocol"].(string)
	eventsubObj.Status.SubscriptionType = eventsubscriptionDetails["SubscriptionType"].(string)
	eventsubObj.Status.Context = eventsubscriptionDetails["Context"].(string)

	if name, ok := eventsubscriptionDetails["Name"].(string); ok {
		eventsubObj.Status.Name = name
	}
	if eventTypes, ok := eventsubscriptionDetails["EventTypes"].([]interface{}); ok {
		eventsubObj.Status.EventTypes = ConvertInterfaceToStringArray(eventTypes)
	}
	if messageIDs, ok := eventsubscriptionDetails["MessageIds"].([]interface{}); ok {
		eventsubObj.Status.MessageIds = ConvertInterfaceToStringArray(messageIDs)
	}

	if resourceTypes, ok := eventsubscriptionDetails["ResourceTypes"].([]interface{}); ok {
		eventsubObj.Status.ResourceTypes = ConvertInterfaceToStringArray(resourceTypes)
	}
	if len(originResources) > 0 {
		eventsubObj.Status.OriginResources = originResources
	}

	err := r.Client.Status().Update(ctx, eventsubObj)
	if err != nil {
		l.LogWithFields(ctx).Errorf("Error: Updating eventsubscription status for %s: %s", eventsubObj.Status.Name, err.Error())
	}
}

// ---------------------------------GET UPDATED OBJECT-----------------------------------
// getUpdatedBmcObject used to get updated BMC object
func (r *CommonReconciler) GetUpdatedBmcObject(ctx context.Context, ns types.NamespacedName, bmcObj *infraiov1.Bmc) {
	err := r.Client.Get(ctx, ns, bmcObj)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching updated %s BMC Object: %s", bmcObj.Spec.BmcDetails.Address, err.Error()))
	}
}

// GetUpdatedOdimObject gets the updated Odim object
func (r *CommonReconciler) GetUpdatedOdimObject(ctx context.Context, ns types.NamespacedName, odimObj *infraiov1.Odim) {
	err := r.Client.Get(ctx, ns, odimObj)
	if err != nil {
		l.LogWithFields(ctx).Error("Error fetching updated Object of ODIM" + err.Error())
	}
}

// GetUpdatedVolumeObject used to get updated BMC object
func (r *CommonReconciler) GetUpdatedVolumeObject(ctx context.Context, ns types.NamespacedName, volObj *infraiov1.Volume) {
	err := r.Client.Get(ctx, ns, volObj)
	if err != nil {
		l.LogWithFields(ctx).Error("Error fetching updated Volume Object:", err.Error())
	}
}

// ----------------------------- DELETE OBJECT --------------------------

// DeleteBmcObject will delete the bmc object from operator
func (r *CommonReconciler) DeleteBmcObject(ctx context.Context, bmcObj *infraiov1.Bmc) {
	err := r.Client.Delete(ctx, bmcObj)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("BMC object %s not deleted", bmcObj.ObjectMeta.Name), err)
	}
	l.LogWithFields(ctx).Info(fmt.Sprintf("BMC object %s deleted successfully", bmcObj.ObjectMeta.Name))
}

// DeleteVolumeObject will delete the volume object from operator
func (r *CommonReconciler) DeleteVolumeObject(ctx context.Context, volObj *infraiov1.Volume) {
	err := r.Client.Delete(ctx, volObj)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Volume object %s not deleted", volObj.Status.VolumeID), err)
	}
	l.LogWithFields(ctx).Info(fmt.Sprintf("Volume object %s deleted successfully", volObj.Status.VolumeID))
}

// GetUpdatedFirmwareObject used to get updated BMC object
func (r *CommonReconciler) GetUpdatedFirmwareObject(ctx context.Context, ns types.NamespacedName, firmObj *infraiov1.Firmware) {
	err := r.Client.Get(ctx, ns, firmObj)
	if err != nil {
		l.LogWithFields(ctx).Error("Error fetching updated Firmware Object:", err.Error())
	}

}

// GetUpdatedEventsubscriptionObjects used to get updated BMC object
func (r *CommonReconciler) GetUpdatedEventsubscriptionObjects(ctx context.Context, ns types.NamespacedName, eventSubObj *infraiov1.Eventsubscription) {
	err := r.Client.Get(ctx, ns, eventSubObj)
	if err != nil {
		l.LogWithFields(ctx).Error("Error fetching updated Eventsubscription Object:", err.Error())
	}
}

// -----------------------------GET OBJECT DETAILS--------------------------
// getSecret returns the username,password,authenticationType for a particular secret
func (r *CommonReconciler) GetObjectSecret(ctx context.Context, secret *corev1.Secret, secretName, rootdir string) (string, string, string) {

	ns := types.NamespacedName{Namespace: config.Data.Namespace, Name: secretName}
	err := r.Client.Get(ctx, ns, secret)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching %s secret", secretName) + err.Error())
	}
	authType := string(secret.Type)
	user := string(secret.Data["username"])
	pass := string(secret.Data["password"])
	return user, pass, authType
}

// getHostPort returns host,port of odim
func GetHostPort(ctx context.Context, uri string) (string, string, error) {
	hostport, err := url.Parse(uri)
	if err != nil {
		l.LogWithFields(ctx).Error("Error: Getting host and port of ODIM" + err.Error())
		return "", "", errors.New("error: Retriving host/port of ODIM")
	}
	host, port, _ := net.SplitHostPort(hostport.Host)
	return host, port, nil
}

// GetEventSubscriptionObjectName will parse and return eventsubscription object name which will be supported by metadata.name
func (r *CommonReconciler) GetEventSubscriptionObjectName(ctx context.Context, destination, name, ns string) string {
	destinationIP := strings.Split(destination, "/")
	var subscriptionObjName string
	if name != "" {
		subscriptionObj := r.GetEventsubscriptionObject(ctx, constants.MetadataName, strings.ToLower(name), ns)
		if subscriptionObj != nil {
			subscriptionObjName = strings.Replace(destinationIP[2], ":", ".", 1)
		} else {
			subscriptionObjName = strings.ToLower(name)
		}
	} else {
		subscriptionObjName = strings.Replace(destinationIP[2], ":", ".", 1)
	}
	return subscriptionObjName
}
