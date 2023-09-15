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
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
)

func Test_odimUtils_updateConnectionMethodVariantsStatus(t *testing.T) {
	type args struct {
		connMethVarInfo map[string]string
	}
	tests := []struct {
		name string
		ou   *odimUtils
		args args
	}{
		{
			name: "Success case",
			ou:   &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{Status: infraiov1.OdimStatus{ConnMethVariants: map[string]string{}}}},
			args: args{connMethVarInfo: map[string]string{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ou.updateConnectionMethodVariantsStatus(tt.args.connMethVarInfo)
		})
	}
}

func Test_odimUtils_checkOdimConnection(t *testing.T) {
	tests := []struct {
		name string
		ou   *odimUtils
		want bool
	}{
		{
			name: "Connected case",
			ou:   &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{url: "connected_case"}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{Status: infraiov1.OdimStatus{ConnMethVariants: map[string]string{}}}},
			want: true,
		},
		{
			name: "Not Connected case",
			ou:   &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{url: "not_connected_case"}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{Status: infraiov1.OdimStatus{ConnMethVariants: map[string]string{}}}},
			want: false,
		},
		{
			name: "Error case",
			ou:   &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{url: "error_case"}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{Status: infraiov1.OdimStatus{ConnMethVariants: map[string]string{}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ou.checkOdimConnection(); got != tt.want {
				t.Errorf("odimUtils.checkOdimConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_odimUtils_updateConnectionMethodVariants(t *testing.T) {
	tests := []struct {
		name string
		ou   *odimUtils
	}{
		{
			name: "Success case",
			ou:   &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{Status: infraiov1.OdimStatus{ConnMethVariants: map[string]string{}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ou.updateConnectionMethodVariants()
		})
	}
}

func Test_getOdimUtils(t *testing.T) {
	ctx := context.TODO()
	odimObj := infraiov1.Odim{}
	commonRec := &mockCommonRec{}
	client := &mockRestClient{}
	commonUtil := common.GetCommonUtils(client)
	type args struct {
		ctx        context.Context
		restClient restclient.RestClientInterface
		commonRec  utils.ReconcilerInterface
		odimObj    *infraiov1.Odim
		commonUtil common.CommonInterface
	}
	tests := []struct {
		name string
		args args
		want odimInterface
	}{
		{
			name: "Success case",
			args: args{ctx: ctx, restClient: client, commonRec: commonRec, odimObj: &odimObj, commonUtil: commonUtil},
			want: &odimUtils{ctx: ctx, odimRestClient: client, commonRec: commonRec, odimObj: &odimObj, commonUtil: commonUtil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOdimUtils(tt.args.ctx, tt.args.restClient, tt.args.commonRec, tt.args.odimObj); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getOdimUtils() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_odimUtils_CreateEventSubscription(t *testing.T) {
	config.Data.OperatorEventSubscriptionEventTypes = []string{"event_type"}
	config.Data.OperatorEventSubscriptionMeesageIds = []string{"msg_id"}
	config.Data.OperatorEventSubscriptionResourceTypes = []string{"resource_type"}
	type args struct {
		destination string
	}
	tests := []struct {
		name    string
		ou      *odimUtils
		args    args
		wantErr bool
	}{
		{
			name:    "Success case",
			ou:      &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{}, commonUtil: mockCommonUtil{}},
			args:    args{destination: "/odim/events"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ou.CreateEventSubscription(tt.args.destination); (err != nil) != tt.wantErr {
				t.Errorf("odimUtils.CreateEventSubscription() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_odimUtils_removeEventsSubscription(t *testing.T) {
	tests := []struct {
		name string
		ou   *odimUtils
		want bool
	}{
		{
			name: "Success case",
			ou:   &odimUtils{ctx: context.TODO(), odimRestClient: &mockRestClient{}, commonRec: &mockCommonRec{}, odimObj: &infraiov1.Odim{}, commonUtil: mockCommonUtil{}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ou.removeEventsSubscription(); got != tt.want {
				t.Errorf("odimUtils.removeEventsSubscription() = %v, want %v", got, tt.want)
			}
		})
	}
}