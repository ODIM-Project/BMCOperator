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
	"reflect"
	"testing"

	common "github.com/ODIM-Project/BMCOperator/controllers/common"
)

func TestPollingReconciler_filterOutVolumesInProgress(t *testing.T) {
	volStatus := common.VolumeStatus{IsGettingCreated: false, IsGettingDeleted: true}
	common.TrackVolumeStatus = map[common.BmcStorageControllerVolumesStatus]common.VolumeStatus{{Bmc: "10.10.10.10", StorageController: "ArrayController-0", VolumeID: "1"}: volStatus}
	type args struct {
		volumeIds         []string
		bmcName           string
		bmcSystemID       string
		storageController string
		operation         string
	}
	tests := []struct {
		name string
		pr   PollingReconciler
		args args
		want []string
	}{
		{
			name: "Success case",
			pr:   PollingReconciler{Client: nil, Scheme: nil, odimObj: nil, bmcObject: nil, ctx: context.TODO(), commonRec: nil, pollRestClient: nil, namespace: "bmc-op"},
			args: args{volumeIds: []string{"1", "2", "3", "4"}, bmcName: "10.10.10.10", bmcSystemID: "somefakesystemid", storageController: "ArrayController-0", operation: "IsGettingDeleted"},
			want: []string{"2", "3", "4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pr.filterOutVolumesInProgress(tt.args.volumeIds, tt.args.bmcName, tt.args.bmcSystemID, tt.args.storageController, tt.args.operation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PollingReconciler.filterOutVolumesInProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPollingReconciler_getVolumes(t *testing.T) {
	type args struct {
		volumeObjectsMap1 map[string][]string
		volumeObjectsMap2 map[string][]string
	}
	tests := []struct {
		name string
		pr   PollingReconciler
		args args
		want map[string][]string
	}{
		{
			name: "Success case",
			pr:   PollingReconciler{Client: nil, Scheme: nil, odimObj: nil, bmcObject: nil, ctx: context.TODO(), commonRec: nil, pollRestClient: nil, namespace: "bmc-op"},
			args: args{volumeObjectsMap1: map[string][]string{"ArrayController-0": {"1", "2", "3", "4"}}, volumeObjectsMap2: map[string][]string{"ArrayController-0": {"1", "2"}}},
			want: map[string][]string{"ArrayController-0": {"3", "4"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pr.getVolumes(tt.args.volumeObjectsMap1, tt.args.volumeObjectsMap2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PollingReconciler.getVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Volumes in Operator = {"ArrayController-0":{"1","2"},"ArrayController-1":{"4","5","6"}}
// Volumes in ODIM = {"ArrayController-0":{"1","2","3"},"ArrayController-1":{"4","5"}}
// Difference: {"ArrayController-0":{"3"}} : Accommodate Add (checkVolumeCreatedAndAccommodate)
//
//	{"ArrayController-1":{"6"}} : Accommodate Delete (checkVolumeDeletedAndAccommodate)
func TestPollingReconciler_AccommodateVolumeDetails(t *testing.T) {
	// TrackVolumeStatus used in filterOutVolumesInProgress
	common.TrackVolumeStatus = map[common.BmcStorageControllerVolumesStatus]common.VolumeStatus{}
	tests := []struct {
		name string
		pr   PollingReconciler
	}{
		{
			name: "Success case",
			pr: PollingReconciler{
				Client:         &MockCommonRec{}, //same struct implements client functions
				Scheme:         nil,
				odimObj:        nil,
				bmcObject:      nil,
				ctx:            context.TODO(),
				commonRec:      &MockCommonRec{},
				pollRestClient: &mockRestClient{},
				commonUtil:     mockCommonUtil{},
				volUtil:        mockVolumeUtil{},
				namespace:      "bmc-op",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.AccommodateVolumeDetails()
		})
	}
}

// Volumes in Operator = {"ArrayController-0":{"1","2"},"ArrayController-1":{"4","5","6"}}
// Volumes in ODIM = {"ArrayController-0":{"1","2","3"},"ArrayController-1":{"4","5"}}
// Difference: {"ArrayController-0":{"3"}} : Revert Add (checkVolumeCreatedAndRevert)
//
//	{"ArrayController-1":{"6"}} : Revert Delete (checkVolumeDeletedAndRevert)
func TestPollingReconciler_RevertVolumeDetails(t *testing.T) {
	// TrackVolumeStatus used in filterOutVolumesInProgress
	common.TrackVolumeStatus = map[common.BmcStorageControllerVolumesStatus]common.VolumeStatus{}
	tests := []struct {
		name string
		pr   PollingReconciler
	}{
		{
			name: "Success case",
			pr: PollingReconciler{
				Client:         &MockCommonRec{}, //same struct implements client functions
				Scheme:         nil,
				odimObj:        nil,
				bmcObject:      nil,
				ctx:            context.TODO(),
				commonRec:      &MockCommonRec{},
				pollRestClient: &mockRestClient{},
				commonUtil:     mockCommonUtil{},
				volUtil:        mockVolumeUtil{},
				bmcUtil:        &mockBmcUtil{},
				namespace:      "bmc-op",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.RevertVolumeDetails()
		})
	}
}
