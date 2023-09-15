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

package controllers

import (
	"context"
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_bootUtils_updateBootSettings(t *testing.T) {
	bootObj := &infraiov1.BootOrderSetting{
		ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
		Spec: infraiov1.BootOrderSettingsSpec{
			SerialNo: "HJDNEY438UH",
			Boot: &infraiov1.BootSetting{
				BootOrder:                    []string{"some", "boot", "order"},
				BootSourceOverrideTarget:     "someTarget",
				BootSourceOverrideEnabled:    "enabled",
				UefiTargetBootSourceOverride: "override",
			},
		},
		Status: infraiov1.BootOrderSettingsStatus{
			Boot: infraiov1.BootSetting{
				BootOrder:                 []string{"some", "boot", "order"},
				BootTargetAllowableValues: []string{"someTarget"},
				UefiTargetAllowableValues: []string{"override"},
			},
		}}
	tests := []struct {
		name string
		bo   *bootUtils
		want bool
	}{
		{
			name: "Success case",
			bo:   &bootUtils{ctx: context.TODO(), bootObj: bootObj, commonRec: &mockCommonRec{}, bootRestClient: &mockRestClient{}, commonUtil: mockCommonUtil{}, namespace: "bmc-op"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bo.updateBootSettings(); got != tt.want {
				t.Errorf("bootUtils.updateBootSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bootUtils_UpdateBootAttributesOnReset(t *testing.T) {
	type args struct {
		bmcName     string
		bootSetting *infraiov1.BootSetting
	}
	tests := []struct {
		name string
		bo   *bootUtils
		args args
	}{
		{
			name: "Succes case",
			bo:   &bootUtils{ctx: context.TODO(), bootObj: nil, commonRec: &mockCommonRec{}, bootRestClient: &mockRestClient{}, commonUtil: mockCommonUtil{}, namespace: "bmc-op"},
			args: args{bmcName: "10.10.10.10", bootSetting: &infraiov1.BootSetting{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.bo.UpdateBootAttributesOnReset(tt.args.bmcName, tt.args.bootSetting)
		})
	}
}

func TestGetBootUtils(t *testing.T) {
	ctx := context.TODO()
	bootObj := infraiov1.BootOrderSetting{}
	commonRec := &mockCommonRec{}
	client := &mockRestClient{}
	util := mockCommonUtil{}
	type args struct {
		ctx            context.Context
		bootObj        *infraiov1.BootOrderSetting
		commonRec      utils.ReconcilerInterface
		bootRestClient restclient.RestClientInterface
		commonUtil     common.CommonInterface
		ns             string
	}
	tests := []struct {
		name string
		args args
		want BootInterface
	}{
		{
			name: "Success case",
			args: args{ctx, &bootObj, commonRec, client, util, "bmc-op"},
			want: &bootUtils{ctx, &bootObj, commonRec, client, util, "bmc-op"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBootUtils(tt.args.ctx, tt.args.bootObj, tt.args.commonRec, tt.args.bootRestClient, tt.args.commonUtil, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBootUtils() = %v, want %v", got, tt.want)
			}
		})
	}
}
