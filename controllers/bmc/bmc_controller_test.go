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
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_bmcUtils_UpdateBmcLabels(t *testing.T) {
	actual := &infraiov1.Bmc{
		Spec:   infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}},
		Status: infraiov1.BmcStatus{BmcSystemID: "bmcFakeID", SerialNumber: "hjketgfdf9", VendorName: "HPE", ModelID: "12 5g6", FirmwareVersion: "1 20 ILO"},
	}
	expected := &infraiov1.Bmc{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			"firmwareVersion": "1.20.ILO",
			"modelId":         "12.5g6",
			"name":            "10.10.10.10",
			"serialNo":        "hjketgfdf9",
			"systemId":        "bmcFakeID",
			"vendor":          "HPE",
		}},
		Spec:   infraiov1.BmcSpec{BmcDetails: infraiov1.BMC{Address: "10.10.10.10"}},
		Status: infraiov1.BmcStatus{BmcSystemID: "bmcFakeID", SerialNumber: "hjketgfdf9", VendorName: "HPE", ModelID: "12 5g6", FirmwareVersion: "1 20 ILO"},
	}
	tests := []struct {
		name string
		bu   *bmcUtils
	}{
		{
			name: "Success case",
			bu:   &bmcUtils{ctx: context.TODO(), bmcObj: actual, namespace: "bmc-op", commonRec: &mockCommonRec{}, bmcRestClient: &mockRestClient{}, commonUtil: nil, updateBMCObject: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.bu.UpdateBmcLabels()
		})
	}
	assert.Equal(t, expected, actual)
}
