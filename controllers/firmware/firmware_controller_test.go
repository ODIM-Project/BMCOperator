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
// under the License..

// Package controllers ...

package controllers

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var mockCR1 = mockCommonRec{variable: "sendBmcObj"}
var mockCR2 = mockCommonRec{variable: "doNotSendBmcObj"}
var mockRC = &mockRestClient{}

func Test_firmwareUtils_getSystemID(t *testing.T) {
	tests := []struct {
		name string
		fu   *firmwareUtils
		want string
	}{
		{
			name: "",
			fu:   &firmwareUtils{context.TODO(), &infraiov1.Firmware{}, &mockCR1, mockRC, nil, nil, "bmc-op"},
			want: "fakeSystemID.1",
		},
		{
			name: "",
			fu:   &firmwareUtils{context.TODO(), &infraiov1.Firmware{}, &mockCR2, mockRC, nil, nil, "bmc-op"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fu.GetSystemID(); got != tt.want {
				t.Errorf("firmwareUtils.GetSystemID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_firmwareUtils_getFirmwareBody(t *testing.T) {

	var body = FirmPayloadWithoutAuth{Image: "fakeimageloc/", Targets: []string{"/redfish/v1/Systems/fakeSystemID.1"}}
	var out, _ = json.Marshal(body)
	tests := []struct {
		name    string
		fu      *firmwareUtils
		want    []byte
		wantErr bool
	}{
		{
			name:    "bmcObj is none",
			fu:      &firmwareUtils{context.TODO(), &infraiov1.Firmware{}, &mockCR2, mockRC, nil, nil, "bmc-op"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "success case",
			fu:      &firmwareUtils{context.TODO(), &infraiov1.Firmware{Spec: infraiov1.FirmwareSpec{Image: infraiov1.ImageDetails{ImageLocation: "fakeimageloc/"}}}, &mockCR1, mockRC, nil, nil, "bmc-op"},
			want:    out,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fu.getFirmwareBody()
			if (err != nil) != tt.wantErr {
				t.Errorf("firmwareUtils.getFirmwareBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("firmwareUtils.getFirmwareBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

// comment the sleep in function and then run testcase
func Test_firmwareUtils_updateFirmwareStatus(t *testing.T) {
	type args struct {
		status        string
		firmVersion   string
		imageLocation string
	}
	tests := []struct {
		name string
		fu   *firmwareUtils
		args args
	}{
		{
			name: "success case",
			fu:   &firmwareUtils{context.TODO(), &infraiov1.Firmware{}, &mockCR1, mockRC, nil, nil, "bmc-op"},
			args: args{status: "Success", firmVersion: "2.60"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fu.UpdateFirmwareStatus(tt.args.status, tt.args.firmVersion, tt.args.imageLocation)
		})
	}
}

func Test_firmwareUtils_GetFirmwareVersion(t *testing.T) {
	tests := []struct {
		name string
		fu   *firmwareUtils
		want string
	}{
		{
			name: "success case",
			fu:   &firmwareUtils{context.TODO(), &infraiov1.Firmware{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{}}}, &mockCR1, &mockRestClient{url: "/redfish/v1/Managers/fakeSystemID.1"}, nil, nil, "bmc-op"},
			want: "2.60",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fu.GetFirmwareVersion(); got != tt.want {
				t.Errorf("firmwareUtils.GetFirmwareVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatForLabels(t *testing.T) {
	type args struct {
		firmVersion string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success case",
			args: args{"ILO v1 2.60"},
			want: "ILO-v1-2.60",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatForLabels(tt.args.firmVersion); got != tt.want {
				t.Errorf("formatForLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

// comment the sleep in function and then run testcase
func Test_firmwareUtils_UpdateFirmwareLabels(t *testing.T) {
	type args struct {
		firmVersion string
	}
	tests := []struct {
		name string
		fu   *firmwareUtils
		args args
	}{
		{
			name: "Success case",
			fu:   &firmwareUtils{context.TODO(), &infraiov1.Firmware{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{}}}, &mockCR1, mockRC, nil, nil, "bmc-op"},
			args: args{firmVersion: "ILO v1 2.60"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fu.UpdateFirmwareLabels(tt.args.firmVersion)
		})
	}
}
