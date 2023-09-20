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
	"encoding/json"
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_biosUtils_getBiosLinkAndBody(t *testing.T) {
	resp := map[string]interface{}{
		"Bios": map[string]interface{}{
			"@odata.id": "/redfish/v1/Systems/fakeID/Bios",
		},
	}
	body := map[string]map[string]interface{}{"Attributes": {"someAttribute": "someValue"}}
	respBody, _ := json.Marshal(body)
	type args struct {
		resp      map[string]interface{}
		biosProps map[string]interface{}
		biosBmcIP string
	}
	tests := []struct {
		name  string
		bs    *biosUtils
		args  args
		want  string
		want1 []byte
	}{
		{
			name:  "Success case",
			bs:    &biosUtils{ctx: context.TODO(), biosObj: &infraiov1.BiosSetting{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}}, commonRec: &mockCommonRec{}, biosRestClient: &mockRestClient{}, commonUtil: nil, namespace: "bmc-op"},
			args:  args{resp, map[string]interface{}{"someAttribute": "someValue"}, "10.10.10.10"},
			want:  "/redfish/v1/Systems/fakeID/Bios/Settings/",
			want1: respBody,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.bs.getBiosLinkAndBody(tt.args.resp, tt.args.biosProps, tt.args.biosBmcIP)
			if got != tt.want {
				t.Errorf("biosUtils.getBiosLinkAndBody() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("biosUtils.getBiosLinkAndBody() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_biosUtils_ValidateBiosAttributes(t *testing.T) {
	type args struct {
		biosID string
	}
	tests := []struct {
		name  string
		bs    *biosUtils
		args  args
		want  bool
		want1 map[string]interface{}
	}{
		{
			name:  "success case",
			bs:    &biosUtils{ctx: context.TODO(), biosObj: &infraiov1.BiosSetting{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}, Spec: infraiov1.BiosSettingSpec{Bios: map[string]string{}}}, commonRec: &mockCommonRec{}, biosRestClient: &mockRestClient{}, commonUtil: nil, namespace: "bmc-op"},
			args:  args{biosID: "BiosAttributeRegistryU32.v1_2_68"},
			want:  true,
			want1: map[string]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.bs.ValidateBiosAttributes(tt.args.biosID)
			if got != tt.want {
				t.Errorf("biosUtils.ValidateBiosAttributes() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("biosUtils.ValidateBiosAttributes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_biosUtils_UpdateBiosAttributesOnReset(t *testing.T) {
	type args struct {
		biosBmcIP             string
		updatedBiosAttributes map[string]string
	}
	tests := []struct {
		name string
		bs   *biosUtils
		args args
	}{
		{
			name: "Success case",
			bs:   &biosUtils{ctx: context.TODO(), biosObj: &infraiov1.BiosSetting{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}, Name: "10.10.10.10"}}, commonRec: &mockCommonRec{}, biosRestClient: &mockRestClient{}, commonUtil: nil, namespace: "bmc-op"},
			args: args{biosBmcIP: "10.10.10.10", updatedBiosAttributes: map[string]string{"someAttribute": "someValue"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.bs.UpdateBiosAttributesOnReset(tt.args.biosBmcIP, tt.args.updatedBiosAttributes)
		})
	}
}

func Test_biosUtils_GetBiosAttributes(t *testing.T) {
	type args struct {
		bmcObj *infraiov1.Bmc
	}
	tests := []struct {
		name string
		bs   *biosUtils
		args args
		want map[string]string
	}{
		{
			name: "success case",
			bs:   &biosUtils{ctx: context.TODO(), biosObj: &infraiov1.BiosSetting{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}, Name: "10.10.10.10"}}, commonRec: &mockCommonRec{}, biosRestClient: &mockRestClient{}, commonUtil: nil, namespace: "bmc-op"},
			args: args{&infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "fakeID"}}},
			want: map[string]string{"someAttribute": "someValue"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bs.GetBiosAttributes(tt.args.bmcObj); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("biosUtils.GetBiosAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_biosUtils_GetBiosAttributeID(t *testing.T) {
	type args struct {
		biosVersion string
		bmcObj      *infraiov1.Bmc
	}
	tests := []struct {
		name  string
		bs    *biosUtils
		args  args
		want  string
		want1 map[string]interface{}
	}{
		{
			name: "Success case",
			bs:   &biosUtils{ctx: context.TODO(), biosObj: &infraiov1.BiosSetting{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}, Name: "10.10.10.10"}}, commonRec: &mockCommonRec{}, biosRestClient: &mockRestClient{}, commonUtil: nil, namespace: "bmc-op"},
			args: args{"U32 v2.68 (07/14/2022)", &infraiov1.Bmc{Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}}},
			want: "BiosAttributeRegistryU32.v1_2_68",
			want1: map[string]interface{}{
				"Id": "BiosAttributeRegistryU32.v1_2_68",
				"SupportedSystems": []interface{}{
					map[string]interface{}{
						"FirmwareVersion": "U32 v2.68 (07/14/2022)",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.bs.GetBiosAttributeID(tt.args.biosVersion, tt.args.bmcObj)
			if got != tt.want {
				t.Errorf("biosUtils.GetBiosAttributeID() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("biosUtils.GetBiosAttributeID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGetBiosUtils(t *testing.T) {
	ctx := context.TODO()
	biosObj := infraiov1.BiosSetting{}
	commonRec := &mockCommonRec{}
	client := &mockRestClient{}
	type args struct {
		ctx            context.Context
		biosObj        *infraiov1.BiosSetting
		commonRec      utils.ReconcilerInterface
		biosRestClient restclient.RestClientInterface
		ns             string
	}
	tests := []struct {
		name string
		args args
		want BiosInterface
	}{
		{
			name: "success case",
			args: args{ctx, &biosObj, commonRec, client, "bmc-op"},
			want: &biosUtils{ctx, &biosObj, commonRec, client, common.GetCommonUtils(client), "bmc-op"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBiosUtils(tt.args.ctx, tt.args.biosObj, tt.args.commonRec, tt.args.biosRestClient, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBiosUtils() = %v, want %v", got, tt.want)
			}
		})
	}
}
