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

// Package controllers ...
package controllers

import (
	"context"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPollingReconciler_AccommodateState(t *testing.T) {
	var bmcObj = infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}, Status: infraiov1.BmcStatus{BmcSystemID: "somefakeid", PowerState: "On"}}
	tests := []struct {
		name string
		pr   *PollingReconciler
	}{
		{
			name: "Success case",
			pr: &PollingReconciler{
				Client:         &MockClient{},
				Scheme:         nil,
				odimObj:        nil,
				bmcObject:      &bmcObj,
				ctx:            context.TODO(),
				commonRec:      &MockCommonRec{},
				pollRestClient: &mockRestClient{},
				commonUtil:     mockCommonUtil{},
				namespace:      "bmc-op",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.AccommodateState()
		})
	}
}

func TestPollingReconciler_RevertState(t *testing.T) {
	var bmcObj = infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}, Status: infraiov1.BmcStatus{BmcSystemID: "somefakeid", PowerState: "On"}}
	tests := []struct {
		name string
		pr   *PollingReconciler
	}{
		{
			name: "Success case",
			pr: &PollingReconciler{
				Client:         &MockClient{},
				Scheme:         nil,
				odimObj:        nil,
				bmcObject:      &bmcObj,
				ctx:            context.TODO(),
				commonRec:      &MockCommonRec{},
				pollRestClient: &mockRestClient{},
				commonUtil:     mockCommonUtil{},
				namespace:      "bmc-op",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.RevertState()
		})
	}
}
