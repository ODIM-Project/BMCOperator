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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_volumeUtils_DeleteVolume(t *testing.T) {
	type args struct {
		bmcObj *infraiov1.Bmc
	}
	tests := []struct {
		name string
		vu   *volumeUtils
		args args
		want bool
	}{
		{
			name: "Success case",
			vu: &volumeUtils{ctx: context.TODO(), volumeRestClient: &mockRestClient{}, commonRec: &mockCommonRec{}, commonUtil: &mockCommonUtil{},
				volObj: &infraiov1.Volume{
					ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10.testvol"},
					Status:     infraiov1.VolumeStatus{StorageControllerID: "ArrayController-0", VolumeID: "1", VolumeName: "testvol"},
				}, namespace: "bmc-op"},
			args: args{bmcObj: &infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10"}, Status: infraiov1.BmcStatus{BmcSystemID: "volFakeID"}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vu.DeleteVolume(tt.args.bmcObj); got != tt.want {
				t.Errorf("volumeUtils.DeleteVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}
