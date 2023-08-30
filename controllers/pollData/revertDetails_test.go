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
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockRestClient is a mock implementation of the RestClientInterface interface
type MockRestClient struct{ mock.Mock }

func (m *MockRestClient) Delete(url string, description string) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusAccepted, Header: http.Header{"Location": []string{"/redfish/v1/TaskMon/1"}}}, nil
}

func (m *MockRestClient) Get(url string, description string) (map[string]interface{}, int, error) {
	return nil, http.StatusNoContent, nil
}

func (m *MockRestClient) Post(url string, description string, body interface{}) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func (m *MockRestClient) Patch(url string, description string, body interface{}) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func (m *MockRestClient) Put(url string, description string, body interface{}) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func (m *MockRestClient) RecallWithNewToken(url, reason, method string, body interface{}) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestPrepareAddBmcPayload(t *testing.T) {
	ctx := context.Background()

	reconciler := PollingReconciler{
		bmcObject: &infraiov1.Bmc{
			Spec: infraiov1.BmcSpec{
				BmcDetails: infraiov1.BMC{
					Address: "example.com",
				},
				Credentials: infraiov1.Credential{
					Username: "admin",
				},
			},
		},
	}

	connectionMethod := "/redfish/v1/ConnectionMethods/1"
	password := "password123"

	expectedPayload := []byte(`{"HostName":"example.com","UserName":"admin","Password":"password123","Links":{"ConnectionMethod":{"@odata.id":"/redfish/v1/ConnectionMethods/1"}}}`)

	payload, err := reconciler.prepareBmcPayload(ctx, connectionMethod, password)
	assert.NoError(t, err)

	assert.Equal(t, expectedPayload, payload)
}

type MockClient struct {
	mock.Mock
}

// Implement the Create method for the mock client
func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Implement the Update method for the mock client
func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

// Implement the Update method for the mock client
func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	//args := m.Called(ctx, obj, opts)
	return nil
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}
