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
	// "fmt"
	"net/http"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	v1 "github.com/ODIM-Project/BMCOperator/api/v1"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	volume "github.com/ODIM-Project/BMCOperator/controllers/volume"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var count = 0

type MockCommonRec struct {
	variable string
}
type mockRestClient struct{ url string }

type mockCommonUtil struct{}

type mockVolumeUtil struct{}

type mockEventSubscriptionUtil struct {
	name string
}

// --------------------------------COMMON RECONCILER MOCKS------------------------------------------
func (m *MockCommonRec) GetBmcObject(ctx context.Context, field, value, ns string) *infraiov1.Bmc {
	if m.variable == "sendBmcObj" {
		return &infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "fakeSystemID"}}
	} else if m.variable == "doNotSendBmcObj" {
		return nil
	}
	return nil
}

func (m *MockCommonRec) GetAllBmcObject(ctx context.Context, ns string) *[]infraiov1.Bmc {
	bmc1 := infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}, Status: infraiov1.BmcStatus{BmcSystemID: "systemID13"}}
	// bmc2 := infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Status: infraiov1.BmcStatus{BmcSystemID: "somefakeid2"}}
	return &[]infraiov1.Bmc{bmc1}
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
	if bmc.ObjectMeta.Name == "10.10.10.10" {
		return map[string][]string{"ArrayControllers-0": []string{"1", "2"}, "ArrayControllers-1": []string{"4", "5", "6"}}
	}
	return nil
}

func (m *MockCommonRec) DeleteVolumeObject(ctx context.Context, volObj *infraiov1.Volume) {}

func (m *MockCommonRec) DeleteBmcObject(ctx context.Context, bmcObj *infraiov1.Bmc) {}

func (m *MockCommonRec) UpdateEventsubscriptionStatus(ctx context.Context, eventsubObj *infraiov1.Eventsubscription, eventsubscriptionDetails map[string]interface{}, originResouces []string) {
}

func (m *MockCommonRec) GetUpdatedEventsubscriptionObjects(ctx context.Context, ns types.NamespacedName, eventSubObj *infraiov1.Eventsubscription) {
}

func (m *MockCommonRec) GetAllVolumeObjects(ctx context.Context, bmcIP, ns string) []*infraiov1.Volume {
	if bmcIP == "10.10.10.10" {
		vol1 := infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10/volume1"}, Status: infraiov1.VolumeStatus{VolumeID: "1", StorageControllerID: "ArrayControllers-0"}}
		vol2 := infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10/volume2"}, Status: infraiov1.VolumeStatus{VolumeID: "2", StorageControllerID: "ArrayControllers-0"}}
		vol3 := infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10/volume3"}, Status: infraiov1.VolumeStatus{VolumeID: "3", StorageControllerID: "ArrayControllers-0"}}
		vol4 := infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10/volume4"}, Status: infraiov1.VolumeStatus{VolumeID: "4", StorageControllerID: "ArrayControllers-1"}}
		vol5 := infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10/volume5"}, Status: infraiov1.VolumeStatus{VolumeID: "5", StorageControllerID: "ArrayControllers-1"}}
		vol6 := infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10/volume6"}, Status: infraiov1.VolumeStatus{VolumeID: "6", StorageControllerID: "ArrayControllers-1"}}
		controllerutil.AddFinalizer(&vol6, volume.VolFinalizer)
		return []*infraiov1.Volume{&vol1, &vol2, &vol3, &vol4, &vol5, &vol6}
	}
	return nil
}

func (m *MockCommonRec) GetAllBiosSchemaRegistryObjects(ctx context.Context, ns string) *[]infraiov1.BiosSchemaRegistry {
	return nil
}

func (m *MockCommonRec) GetVolumeObjectByVolumeID(ctx context.Context, volumeID, ns string) *infraiov1.Volume {
	if volumeID == "6" {
		volObj := &infraiov1.Volume{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10.Volume6"},
			Spec: infraiov1.VolumeSpec{
				RAIDType: "RAID0",
				Drives:   []string{"6"},
			}}
		return volObj
	}
	return nil
}

// common reconciler funcs
func (m *MockCommonRec) GetCommonReconcilerClient() client.Client {
	return MockCommonRec{}
}

func (m *MockCommonRec) GetCommonReconcilerScheme() *runtime.Scheme {
	return nil
}

// ----------------------------CLIENT MOCKS--------------------------------
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

// --------------------------REST CALL MOCKS---------------------------------
func (m *mockRestClient) Post(url string, reason string, body interface{}) (*http.Response, error) {
	// Test_RevertState : success case
	if url == "/redfish/v1/Systems/somefakeid/Actions/ComputerSystem.Reset" {
		return &http.Response{StatusCode: 202, Header: http.Header{}}, nil
	}
	return nil, nil
}
func (m *mockRestClient) Get(url string, reason string) (map[string]interface{}, int, error) {
	// Test_AccommodateVolumeDetails : success case
	// Accomodate Addition for ArrayControllers-0 Accommodate Deletion for ArrayControllers-1
	// getAllVolumeObjectIdsFromODIM : getting all storage details
	if url == "/redfish/v1/Systems/systemID13/Storage/" {
		members := []interface{}{
			map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-0",
			}, map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1",
			}}
		return map[string]interface{}{"Members": members}, 200, nil
		// return map[string]interface{}{"ArrayControllers-0": []string{"1", "2", "3"}, "ArrayControllers-1": []string{"4", "5"}}, 200, nil
	}
	// getVolumeDetailsAndCreateVolumeObject : creating volume object3, hence filling its details from ODIM
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-0/Volumes/3" {
		newVolume := map[string]interface{}{
			"Name":          "Volume3",
			"RAIDType":      "RAID0",
			"CapacityBytes": float64(10234567),
			"Id":            "3",
			"Identifiers":   []interface{}{map[string]interface{}{"DurableName": "someDurableName", "DurableNameFormat": "someDurableFormat"}},
			"Links": map[string]interface{}{
				"Drives": []interface{}{
					map[string]interface{}{"@odata.id": "someOdataIDDrive/3"},
					map[string]interface{}{"@odata.id": "someOdataIDDrive/2"},
				},
			},
		}
		return newVolume, 200, nil
	}
	// getAllVolumeObjectIdsFromODIM : getting all volume details
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-0/Volumes" {
		members := []interface{}{
			map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-0/Volumes/1",
			}, map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-0/Volumes/2",
			}, map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-0/Volumes/3",
			},
		}
		return map[string]interface{}{"Members": members}, 200, nil
	}
	// get volumes request from getNewVolumeIDForObject for revert case should send back  volume 6 as well, since we have created it
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes" && reason == "Fetching all volumes.." {
		members := []interface{}{
			map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/4",
			}, map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/5",
			}, map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/6",
			},
		}
		return map[string]interface{}{"Members": members}, 200, nil
	}
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes" {
		count++
		members := []interface{}{
			map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/4",
			}, map[string]interface{}{
				"@odata.id": "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/5",
			},
		}
		return map[string]interface{}{"Members": members}, 200, nil
	}
	// Test_RevertVolumeDetails : success case
	// getNewVolumeIDForObject : for creation of volume 6
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/6" {
		newVolume := map[string]interface{}{
			"Name":          "Volume6",
			"RAIDType":      "RAID0",
			"CapacityBytes": float64(10234567),
			"Id":            "6",
			"Identifiers":   []interface{}{map[string]interface{}{"DurableName": "someDurableName", "DurableNameFormat": "someDurableFormat"}},
			"Links": map[string]interface{}{
				"Drives": []interface{}{
					map[string]interface{}{"@odata.id": "someOdataIDDrive/6"},
				},
			},
		}
		return newVolume, 200, nil
	}
	// getNewVolumeIDForObject : for looping
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/5" {
		newVolume := map[string]interface{}{
			"Name":          "Volume5",
			"RAIDType":      "RAID0",
			"CapacityBytes": float64(10234567),
			"Id":            "5",
			"Identifiers":   []interface{}{map[string]interface{}{"DurableName": "someDurableName", "DurableNameFormat": "someDurableFormat"}},
			"Links": map[string]interface{}{
				"Drives": []interface{}{
					map[string]interface{}{"@odata.id": "someOdataIDDrive/1"},
					map[string]interface{}{"@odata.id": "someOdataIDDrive/2"},
				},
			},
		}
		return newVolume, 200, nil
	}
	// getNewVolumeIDForObject : for looping
	if url == "/redfish/v1/Systems/systemID13/Storage/ArrayControllers-1/Volumes/4" {
		newVolume := map[string]interface{}{
			"Name":          "Volume4",
			"RAIDType":      "RAID0",
			"CapacityBytes": float64(10234567),
			"Id":            "4",
			"Identifiers":   []interface{}{map[string]interface{}{"DurableName": "someDurableName", "DurableNameFormat": "someDurableFormat"}},
			"Links": map[string]interface{}{
				"Drives": []interface{}{
					map[string]interface{}{"@odata.id": "someOdataIDDrive/3"},
					map[string]interface{}{"@odata.id": "someOdataIDDrive/4"},
				},
			},
		}
		return newVolume, 200, nil
	}
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

// --------------------------------------EVENT SUBCRIPTION MOCKS-------------------------------
func (m *mockEventSubscriptionUtil) CreateEventsubscription(eventSubObj *infraiov1.Eventsubscription, namespacedName types.NamespacedName) (bool, error) {
	return true, nil
}
func (m *mockEventSubscriptionUtil) UpdateEventsubscriptionStatus(eventSubscriptionID string) bool {
	return true
}
func (m *mockEventSubscriptionUtil) GetEventSubscriptionRequest() ([]byte, error) {
	return nil, nil
}
func (m *mockEventSubscriptionUtil) DeleteEventsubscription() bool {
	return true
}
func (m *mockEventSubscriptionUtil) GetSystemIDFromURI(resourceURIArr []string, resourceOID string) string {
	return ""
}
func (m *mockEventSubscriptionUtil) UpdateEventSubscriptionLabels() {}
func (m *mockEventSubscriptionUtil) MapOriginResources(subscriptionDetails map[string]interface{}) ([]string, error) {
	return []string{}, nil
}

func (m *MockCommonRec) GetRootCAFromSecret(ctx context.Context) []byte {
	return []byte{}
}
func (m *mockEventSubscriptionUtil) GetMessageRegistryDetails(bmcObj *infraiov1.Bmc, registryName string) ([]string, []map[string]interface{}) {
	return []string{}, nil
}
func (m *mockEventSubscriptionUtil) ValidateMessageIDs(messageIDs []string) error {
	return nil
}

// --------------------------COMMON UTIL MOCKS------------------------
func (m mockCommonUtil) GetBmcSystemDetails(context.Context, *infraiov1.Bmc) map[string]interface{} {
	return map[string]interface{}{"PowerState": "Off"}
}
func (m mockCommonUtil) MoniteringTaskmon(headerInfo http.Header, ctx context.Context, operation, resourceName string) (bool, map[string]interface{}) {
	return true, nil
}
func (m mockCommonUtil) BmcAddition(ctx context.Context, bmcObject *v1.Bmc, body []byte, restClient restclient.RestClientInterface) (bool, map[string]interface{}) {
	return false, nil
}
func (m mockCommonUtil) BmcDeleteOperation(ctx context.Context, aggregationURL string, restClient restclient.RestClientInterface, resource string) bool {
	return false
}

// -----------------------------------VOLUME UTIL MOCKS--------------------------------------------------
func (m mockVolumeUtil) CreateVolume(bmcObj *infraiov1.Bmc, dispName string, namespacedName types.NamespacedName, updateVolumeObject bool, commonRec utils.ReconcilerInterface) (bool, error) {
	//successfully created volume
	return true, nil
}
func (m mockVolumeUtil) GetVolumeRequestPayload(systemID, dispName string) []byte {
	return nil
}
func (m mockVolumeUtil) UpdateVolumeStatusAndClearSpec(bmcObj *infraiov1.Bmc, dispName string) (bool, string) {
	return false, ""
}
func (m mockVolumeUtil) DeleteVolume(bmcObj *infraiov1.Bmc) bool {
	//successfully deleted volume
	return true
}
func (m mockVolumeUtil) UpdateVolumeObject(volObj *infraiov1.Volume) {}
