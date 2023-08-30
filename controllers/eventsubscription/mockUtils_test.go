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

type MockCommonRec struct {
	variable string
}
type mockRestClient struct{ url string }

// Mock Common Rec Methods
// get objects
func (m *MockCommonRec) GetBmcObject(ctx context.Context, field, value, ns string) *infraiov1.Bmc {
	if m.variable == "sendBmcObj" {
		return &infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "fakeSystemID"}}
	} else if m.variable == "doNotSendBmcObj" {
		return nil
	}
	return nil
}

func (m *MockCommonRec) GetAllBmcObject(ctx context.Context, ns string) *[]infraiov1.Bmc {
	return nil
}

func (m *MockCommonRec) GetBiosSchemaObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSchemaRegistry {
	return nil
}
func (m *MockCommonRec) GetBiosObject(ctx context.Context, field, value, ns string) *infraiov1.BiosSetting {
	return nil
}

func (m *MockCommonRec) GetBootObject(ctx context.Context, field, value, ns string) *infraiov1.BootOrderSetting {
	return nil
}

func (m *MockCommonRec) GetOdimObject(ctx context.Context, field, value, ns string) *infraiov1.Odim {
	return nil
}

func (m *MockCommonRec) GetVolumeObject(ctx context.Context, bmcIP, ns string) *infraiov1.Volume {
	return nil
}

func (m *MockCommonRec) GetFirmwareObject(ctx context.Context, field, value, ns string) *infraiov1.Firmware {
	return nil
}

// create objects
func (m *MockCommonRec) CreateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, bmcObj *infraiov1.Bmc) bool {
	return true
}

func (m *MockCommonRec) CreateBootOrderSettingObject(ctx context.Context, bootAttributes *infraiov1.BootSetting, bmcObj *infraiov1.Bmc) bool {
	return true
}

func (m *MockCommonRec) CreateEventSubscriptionObject(ctx context.Context, subscriptionDetails map[string]interface{}, ns string, originResources []string) bool {
	return true
}

func (m *MockCommonRec) GetAllEventSubscriptionObjects(ctx context.Context, ns string) *[]infraiov1.Eventsubscription {
	return nil
}

func (m *MockCommonRec) GetEventsubscriptionObject(ctx context.Context, field, value, ns string) *infraiov1.Eventsubscription {
	return nil
}

func (m *MockCommonRec) GetEventMessageRegistryObject(ctx context.Context, registryPrefix, registryVersion, ns string) *infraiov1.EventsMessageRegistry {
	return nil
}

func (m *MockCommonRec) CheckAndCreateEventMessageObject(ctx context.Context, messageRegistryResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool {
	return true
}

func (m *MockCommonRec) CheckAndCreateBiosSchemaObject(ctx context.Context, attributeResp map[string]interface{}, bmcObj *infraiov1.Bmc) bool {
	return true
}

// update objects
func (m *MockCommonRec) UpdateBmcStatus(ctx context.Context, bmcObj *infraiov1.Bmc) {
}

func (m *MockCommonRec) UpdateOdimStatus(ctx context.Context, status string, odimObj *infraiov1.Odim) {
}

func (m *MockCommonRec) UpdateBmcObjectOnReset(ctx context.Context, bmcObject *infraiov1.Bmc, status string) {
}

func (m *MockCommonRec) UpdateVolumeStatus(ctx context.Context, volObject *infraiov1.Volume, volumeID, volumeName, capBytes, durableName, durableNameFormat string) {
}

// get updated objects
func (m *MockCommonRec) GetUpdatedBmcObject(ctx context.Context, ns types.NamespacedName, bmcObj *infraiov1.Bmc) {
}

func (m *MockCommonRec) GetUpdatedOdimObject(ctx context.Context, ns types.NamespacedName, odimObj *infraiov1.Odim) {
}

func (m *MockCommonRec) GetUpdatedVolumeObject(ctx context.Context, ns types.NamespacedName, volObj *infraiov1.Volume) {
}

func (m *MockCommonRec) GetUpdatedFirmwareObject(ctx context.Context, ns types.NamespacedName, firmObj *infraiov1.Firmware) {
}

func (m *MockCommonRec) UpdateBiosSettingObject(ctx context.Context, biosAttributes map[string]string, biosObj *infraiov1.BiosSetting) bool {
	return true
}

func (m *MockCommonRec) GetAllVolumeObjectIds(ctx context.Context, bmc *infraiov1.Bmc, ns string) map[string][]string {
	return nil
}

func (m *MockCommonRec) DeleteVolumeObject(ctx context.Context, volObj *infraiov1.Volume) {}

func (m *MockCommonRec) DeleteBmcObject(ctx context.Context, bmcObj *infraiov1.Bmc) {}

func (m *MockCommonRec) UpdateEventsubscriptionStatus(ctx context.Context, eventsubObj *infraiov1.Eventsubscription, eventsubscriptionDetails map[string]interface{}, originResouces []string) {
}

func (m *MockCommonRec) GetUpdatedEventsubscriptionObjects(ctx context.Context, ns types.NamespacedName, eventSubObj *infraiov1.Eventsubscription) {
}

func (m *MockCommonRec) GetAllVolumeObjects(ctx context.Context, bmcIP, ns string) []*infraiov1.Volume {
	return nil
}

func (m *MockCommonRec) GetAllBiosSchemaRegistryObjects(ctx context.Context, ns string) *[]infraiov1.BiosSchemaRegistry {
	return nil
}

func (m *MockCommonRec) GetVolumeObjectByVolumeID(ctx context.Context, volumeName, ns string) *infraiov1.Volume {
	return nil
}

// common reconciler funcs
func (m *MockCommonRec) GetCommonReconcilerClient() client.Client {
	return MockCommonRec{}
}

func (m *MockCommonRec) GetCommonReconcilerScheme() *runtime.Scheme {
	return nil
}

// Client mocks
func (m MockCommonRec) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return nil
}

func (m MockCommonRec) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (m MockCommonRec) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (m MockCommonRec) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (m MockCommonRec) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (m MockCommonRec) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (m MockCommonRec) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (m MockCommonRec) Status() client.StatusWriter {
	return MockCommonRec{}
}

func (m MockCommonRec) RESTMapper() meta.RESTMapper {
	return nil
}

func (m MockCommonRec) Scheme() *runtime.Scheme {
	return nil
}

func (m MockCommonRec) GetRootCAFromSecret(ctx context.Context) []byte {
	return []byte{}
}

// mock Rest calls

func (m *mockRestClient) Post(string, string, interface{}) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) Get(string, string) (map[string]interface{}, int, error) {
	return nil, 0, nil
}
func (m *mockRestClient) Patch(string, string, interface{}) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) Delete(string, string) (*http.Response, error) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
	}
	return resp, nil
}
func (m *mockRestClient) Put(string, string, interface{}) (*http.Response, error) {
	return nil, nil
}
func (m *mockRestClient) RecallWithNewToken(url, reason, method string, body interface{}) (*http.Response, error) {
	return nil, nil
}
