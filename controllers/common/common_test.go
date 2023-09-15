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
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCommonUtils_GetBmcSystemDetails(t *testing.T) {
	type args struct {
		ctx    context.Context
		bmcObj *infraiov1.Bmc
	}
	tests := []struct {
		name string
		bu   *CommonUtils
		args args
		want map[string]interface{}
	}{
		{
			name: "Success case",
			bu:   &CommonUtils{&mockRestClient{url: "success_case"}},
			args: args{context.TODO(), &infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "commonFakeID"}}},
			want: map[string]interface{}{},
		},
		{
			name: "StatusNotFound case",
			bu:   &CommonUtils{&mockRestClient{url: "StatusNotFound_case"}},
			args: args{context.TODO(), &infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "commonFakeID"}}},
			want: nil,
		},
		{
			name: "Error case",
			bu:   &CommonUtils{&mockRestClient{url: "error_case"}},
			args: args{context.TODO(), &infraiov1.Bmc{Status: infraiov1.BmcStatus{BmcSystemID: "commonFakeID"}}},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bu.GetBmcSystemDetails(tt.args.ctx, tt.args.bmcObj); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonUtils.GetBmcSystemDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}

// comment time.Sleep before testing
func TestCommonUtils_MoniteringTaskmon(t *testing.T) {
	type args struct {
		headerInfo   http.Header
		ctx          context.Context
		operation    string
		resourceName string
	}
	tests := []struct {
		name  string
		bu    *CommonUtils
		args  args
		want  bool
		want1 map[string]interface{}
	}{
		{
			name:  "StatusOK case",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/statusOK"}}, ctx: context.TODO(), operation: ADDBMC, resourceName: "10.10.10.10"},
			want:  true,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusCreated case and addbmc",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusCreated"}}, ctx: context.TODO(), operation: ADDBMC, resourceName: "10.10.10.10"},
			want:  true,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusNotFound case",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusNotFound"}}, ctx: context.TODO(), operation: ADDBMC, resourceName: "10.10.10.10"},
			want:  false,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusBadRequest case",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusBadRequest"}}, ctx: context.TODO(), operation: ADDBMC, resourceName: "10.10.10.10"},
			want:  false,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusConflict case",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusConflict"}}, ctx: context.TODO(), operation: ADDBMC, resourceName: "10.10.10.10"},
			want:  false,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusNotFound case and delete bmc",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusNotFound"}}, ctx: context.TODO(), operation: DELETEBMC, resourceName: "10.10.10.10"},
			want:  true,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusNoContent case and delete bmc",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusNoContent"}}, ctx: context.TODO(), operation: DELETEBMC, resourceName: "10.10.10.10"},
			want:  true,
			want1: map[string]interface{}{},
		},
		{
			name:  "StatusCreated case and event subcription",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/StatusCreated"}}, ctx: context.TODO(), operation: EVENTSUBSCRIPTION, resourceName: "10.10.10.10"},
			want:  true,
			want1: map[string]interface{}{},
		},
		{
			name:  "Error case",
			bu:    &CommonUtils{&mockRestClient{}},
			args:  args{headerInfo: http.Header{"Location": []string{"/taskmon/error"}}, ctx: context.TODO(), operation: ADDBMC, resourceName: "10.10.10.10"},
			want:  false,
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.bu.MoniteringTaskmon(tt.args.headerInfo, tt.args.ctx, tt.args.operation, tt.args.resourceName)
			if got != tt.want {
				t.Errorf("CommonUtils.MoniteringTaskmon() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("CommonUtils.MoniteringTaskmon() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
func TestCommonUtils_BmcAddition(t *testing.T) {
	type args struct {
		ctx        context.Context
		bmcObject  *infraiov1.Bmc
		body       []byte
		restClient restclient.RestClientInterface
	}
	tests := []struct {
		name  string
		bu    *CommonUtils
		args  args
		want  bool
		want1 map[string]interface{}
	}{
		{
			name:  "Error case",
			bu:    &CommonUtils{&mockRestClient{url: "error_case"}},
			args:  args{context.TODO(), &infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}}, []byte{}, &mockRestClient{}},
			want:  false,
			want1: nil,
		},
		{
			name:  "StatusUnauthorized case",
			bu:    &CommonUtils{&mockRestClient{url: "StatusUnauthorized"}},
			args:  args{context.TODO(), &infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}}, []byte{}, &mockRestClient{}},
			want:  false,
			want1: nil,
		},
		{
			name:  "StatusAccepted case",
			bu:    &CommonUtils{&mockRestClient{url: "StatusAccepted"}},
			args:  args{context.TODO(), &infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Spec: infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}}}, []byte{}, &mockRestClient{}},
			want:  true,
			want1: map[string]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.bu.BmcAddition(tt.args.ctx, tt.args.bmcObject, tt.args.body)
			if got != tt.want {
				t.Errorf("CommonUtils.BmcAddition() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("CommonUtils.BmcAddition() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCommonUtils_BmcDeleteOperation(t *testing.T) {
	type args struct {
		ctx            context.Context
		aggregationURL string
		resourceName   string
	}
	tests := []struct {
		name string
		bu   *CommonUtils
		args args
		want bool
	}{
		{
			name: "Error case",
			bu:   &CommonUtils{&mockRestClient{url: "error_case"}},
			args: args{ctx: context.TODO(), aggregationURL: "", resourceName: "10.10.10.10"},
			want: false,
		},
		{
			name: "StatusAccepted case",
			bu:   &CommonUtils{&mockRestClient{url: "StatusAccepted_case"}},
			args: args{ctx: context.TODO(), aggregationURL: "", resourceName: "10.10.10.10"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bu.BmcDeleteOperation(tt.args.ctx, tt.args.aggregationURL, tt.args.resourceName); got != tt.want {
				t.Errorf("CommonUtils.BmcDeleteOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

// comment time.Sleep before testing
func TestBiosAttributeUpdation(t *testing.T) {
	type args struct {
		ctx      context.Context
		body     map[string]interface{}
		systemID string
		client   restclient.RestClientInterface
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Success case",
			args: args{ctx: context.TODO(), body: map[string]interface{}{"attr": "value"}, systemID: "commonFakeID", client: &mockRestClient{}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BiosAttributeUpdation(tt.args.ctx, tt.args.body, tt.args.systemID, tt.args.client); got != tt.want {
				t.Errorf("BiosAttributeUpdation() = %v, want %v", got, tt.want)
			}
		})
	}
}