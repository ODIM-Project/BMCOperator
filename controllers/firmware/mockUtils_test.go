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
	"net/http"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockCommonRec struct {
	variable string
}
type mockRestClient struct{ url string }

// Mock Common Rec Methods
// get objects
func (m *mockCommonRec) GetBmcObject(ctx context.Context, field, value, ns string) *infraiov1.Bmc {
	if m.variable == "sendBmcObj" {
		return &infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "fakeSystemID.1"}}
	} else if m.variable == "doNotSendBmcObj" {
		return nil
	}
	return nil
}

func (m *mockCommonRec) GetAllBmcObject(ctx context.Context, ns string) *[]infraiov1.Bmc {
	return nil
}

func (m *mockCommonRec) GetBiosSchemaObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSchemaRegistry {
	return nil
}
func (m *mockCommonRec) GetBiosObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSetting {
	return nil
}

func (m *mockCommonRec) GetBootObject(ctx context.Context, field, value, ns string) *infraiov1.BootOrderSetting {
	return nil
}

func (m *mockCommonRec) GetOdimObject(ctx context.Context, field, value, ns string) *infraiov1.Odim {
	return nil
}

func (m *mockCommonRec) GetVolumeObject(ctx context.Context, bmcIP, ns string) *infraiov1.Volume {
	return nil
}

func (m *mockCommonRec) GetFirmwareObject(ctx context.Context, field, value, ns string) *infraiov1.Firmware {
	return nil
}

// create objects
func (m *mockCommonRec) CreateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, bmcObj *infraiov1.Bmc) bool {
	return true
}

func (m *mockCommonRec) CreateBootOrderSettingObject(ctx context.Context, bootAttributes *infraiov1.BootSetting, bmcObj *infraiov1.Bmc) bool {
	return true
}

func (m *mockCommonRec) CheckAndCreateBiosSchemaObject(ctx context.Context, attributeResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool {
	return true
}

func (m *mockCommonRec) CreateEventSubscriptionObject(ctx context.Context, subscriptionDetails map[string]interface{}, ns string, originResources []string) bool {
	return true
}

func (m *mockCommonRec) GetAllEventSubscriptionObjects(ctx context.Context, ns string) *[]infraiov1.Eventsubscription {
	return nil
}

func (m *mockCommonRec) GetEventsubscriptionObject(ctx context.Context, field, value, ns string) *infraiov1.Eventsubscription {
	return nil
}

func (m *mockCommonRec) GetEventMessageRegistryObject(ctx context.Context, registryPrefix, registryVersion, ns string) *infraiov1.EventsMessageRegistry {
	return nil
}

func (m *mockCommonRec) CheckAndCreateEventMessageObject(ctx context.Context, messageRegistryResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool {
	return true
}

// update objects
func (m *mockCommonRec) UpdateBmcStatus(ctx context.Context, bmcObj *infraiov1.Bmc) {
}

func (m *mockCommonRec) UpdateOdimStatus(ctx context.Context, status string, odimObj *infraiov1.Odim) {
}

func (m *mockCommonRec) UpdateBmcObjectOnReset(ctx context.Context, bmcObject *infraiov1.Bmc, status string) {
}

func (m *mockCommonRec) UpdateVolumeStatus(ctx context.Context, volObject *infraiov1.Volume, volumeID, volumeName, capBytes, durableName, durableNameFormat string) {
}

// get updated objects
func (m *mockCommonRec) GetUpdatedBmcObject(ctx context.Context, ns types.NamespacedName, bmcObj *infraiov1.Bmc) {
}

func (m *mockCommonRec) GetUpdatedOdimObject(ctx context.Context, ns types.NamespacedName, odimObj *infraiov1.Odim) {
}

func (m *mockCommonRec) GetUpdatedVolumeObject(ctx context.Context, ns types.NamespacedName, volObj *infraiov1.Volume) {
}

func (m *mockCommonRec) GetUpdatedFirmwareObject(ctx context.Context, ns types.NamespacedName, firmObj *infraiov1.Firmware) {
}

func (m *mockCommonRec) UpdateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, biosObj *infraiov1.BiosSetting) bool {
	return true
}

func (m *mockCommonRec) GetAllVolumeObjectIds(ctx context.Context, bmc *infraiov1.Bmc, ns string) map[string][]string {
	return nil
}

func (m *mockCommonRec) GetAllBiosSchemaRegistryObjects(ctx context.Context, ns string) *[]infraiov1.BiosSchemaRegistry {
	return nil
}

func (m *mockCommonRec) DeleteVolumeObject(ctx context.Context, volObj *infraiov1.Volume) {}

func (m *mockCommonRec) DeleteBmcObject(ctx context.Context, bmcObj *infraiov1.Bmc) {}

func (m *mockCommonRec) UpdateEventsubscriptionStatus(ctx context.Context, eventsubObj *infraiov1.Eventsubscription, eventsubscriptionDetails map[string]interface{}, originResouces []string) {
}

func (m *mockCommonRec) GetUpdatedEventsubscriptionObjects(ctx context.Context, ns types.NamespacedName, eventSubObj *infraiov1.Eventsubscription) {
}

func (m *mockCommonRec) GetAllVolumeObjects(ctx context.Context, bmcIP, ns string) []*infraiov1.Volume {
	return nil
}

func (m *mockCommonRec) GetVolumeObjectByVolumeID(ctx context.Context, volumeName, ns string) *infraiov1.Volume {
	return nil
}

func (m *mockCommonRec) GetRootCAFromSecret(ctx context.Context) []byte {
	return []byte{}
}

// common reconciler funcs
func (m *mockCommonRec) GetCommonReconcilerClient() client.Client {
	return mockCommonRec{}
}

func (m *mockCommonRec) GetCommonReconcilerScheme() *runtime.Scheme {
	return nil
}

// Client mocks
func (m mockCommonRec) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return nil
}

func (m mockCommonRec) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (m mockCommonRec) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (m mockCommonRec) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (m mockCommonRec) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (m mockCommonRec) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (m mockCommonRec) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (m mockCommonRec) Status() client.StatusWriter {
	return mockCommonRec{}
}

func (m mockCommonRec) RESTMapper() meta.RESTMapper {
	return nil
}

func (m mockCommonRec) Scheme() *runtime.Scheme {
	return nil
}

// mock Rest calls

func (m *mockRestClient) Post(string, string, interface{}) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) Get(url string, reason string) (map[string]interface{}, int, error) {
	// Test_firmwareUtils_getFirmwareVersion : success case
	if m.url == "/redfish/v1/Managers/fakeSystemID.1" && url == "/redfish/v1/UpdateService/FirmwareInventory" {
		return map[string]interface{}{"Members": []interface{}{map[string]interface{}{"@odata.id": "/redfish/v1/UpdateService/FirmwareInventory/fakeSystemID.5"}}}, 200, nil
	}
	if m.url == "/redfish/v1/Managers/fakeSystemID.1" {
		return map[string]interface{}{"FirmwareVersion": "2.60"}, 200, nil
	}
	return nil, 0, nil
}
func (m *mockRestClient) Patch(string, string, interface{}) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) Delete(string, string) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) Put(string, string, interface{}) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) RecallWithNewToken(url, reason, method string, body interface{}) (*http.Response, error) {
	return nil, nil
}
