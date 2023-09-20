// (C) Copyright [2022] Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package controllers

import (
	"context"
	"reflect"
	"testing"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCommonReconciler_GetBmcObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.Bmc
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetBmcObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetBmcObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetAllBmcObject(t *testing.T) {
	type args struct {
		ctx context.Context
		ns  string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *[]infraiov1.Bmc
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetAllBmcObject(tt.args.ctx, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetAllBmcObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetBiosSchemaObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.BiosSchemaRegistry
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetBiosSchemaObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetBiosSchemaObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetBiosObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.BiosSetting
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetBiosObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetBiosObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetBootObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.BootOrderSetting
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetBootObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetBootObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetOdimObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.Odim
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetOdimObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetOdimObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetVolumeObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		bmcIP string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.Volume
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), bmcIP: "10.10.10.10", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetVolumeObject(tt.args.ctx, tt.args.bmcIP, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetVolumeObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetAllVolumeObjects(t *testing.T) {
	type args struct {
		ctx   context.Context
		bmcIP string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want []*infraiov1.Volume
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), bmcIP: "10.10.10.10", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetAllVolumeObjects(tt.args.ctx, tt.args.bmcIP, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetAllVolumeObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetVolumeObjectByVolumeID(t *testing.T) {
	type args struct {
		ctx      context.Context
		volumeID string
		ns       string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.Volume
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), volumeID: "1", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetVolumeObjectByVolumeID(tt.args.ctx, tt.args.volumeID, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetVolumeObjectByVolumeID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetAllVolumeObjectIds(t *testing.T) {
	type args struct {
		ctx context.Context
		bmc *infraiov1.Bmc
		ns  string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want map[string][]string
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), bmc: &infraiov1.Bmc{}, ns: "bmc-op"},
			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetAllVolumeObjectIds(tt.args.ctx, tt.args.bmc, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetAllVolumeObjectIds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetFirmwareObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.Firmware
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetFirmwareObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetFirmwareObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetEventsubscriptionObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.Eventsubscription
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetEventsubscriptionObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetEventsubscriptionObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetAllEventSubscriptionObjects(t *testing.T) {
	type args struct {
		ctx context.Context
		ns  string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *[]infraiov1.Eventsubscription
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetAllEventSubscriptionObjects(tt.args.ctx, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetAllEventSubscriptionObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetAllBiosSchemaRegistryObjects(t *testing.T) {
	type args struct {
		ctx context.Context
		ns  string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *[]infraiov1.BiosSchemaRegistry
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetAllBiosSchemaRegistryObjects(tt.args.ctx, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetAllBiosSchemaRegistryObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_GetEventMessageRegistryObject(t *testing.T) {
	type args struct {
		ctx   context.Context
		field string
		value string
		ns    string
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want *infraiov1.EventsMessageRegistry
	}{
		{
			name: "List empty case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), field: "aaa", value: "bbb", ns: "bmc-op"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetEventMessageRegistryObject(tt.args.ctx, tt.args.field, tt.args.value, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonReconciler.GetEventMessageRegistryObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_CreateBiosSettingObject(t *testing.T) {
	type args struct {
		ctx            context.Context
		biosAttributes map[string]string
		bmcObj         *infraiov1.Bmc
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want bool
	}{
		{
			name: "Success case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), biosAttributes: map[string]string{"aaa": "bbb"}, bmcObj: &infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10", Namespace: "bmc-op"}, Status: infraiov1.BmcStatus{BmcSystemID: "objectFakeID"}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.CreateBiosSettingObject(tt.args.ctx, tt.args.biosAttributes, tt.args.bmcObj); got != tt.want {
				t.Errorf("CommonReconciler.CreateBiosSettingObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_CreateBootOrderSettingObject(t *testing.T) {
	type args struct {
		ctx            context.Context
		bootAttributes *infraiov1.BootSetting
		bmcObj         *infraiov1.Bmc
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want bool
	}{
		{
			name: "Success case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(), bootAttributes: &infraiov1.BootSetting{}, bmcObj: &infraiov1.Bmc{ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10", Namespace: "bmc-op"}, Status: infraiov1.BmcStatus{BmcSystemID: "objectFakeID"}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.CreateBootOrderSettingObject(tt.args.ctx, tt.args.bootAttributes, tt.args.bmcObj); got != tt.want {
				t.Errorf("CommonReconciler.CreateBootOrderSettingObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonReconciler_CheckAndCreateBiosSchemaObject(t *testing.T) {
	type args struct {
		ctx           context.Context
		attributeResp map[string]interface{}
		bmcObj        *infraiov1.Bmc
	}
	tests := []struct {
		name string
		r    *CommonReconciler
		args args
		want bool
	}{
		{
			name: "Success case",
			r:    &CommonReconciler{&mockClient{}, nil},
			args: args{ctx: context.TODO(),
				attributeResp: map[string]interface{}{
					"Id":           "123",
					"Name":         "test",
					"OwningEntity": "anch",
					"SupportedSystems": []interface{}{
						map[string]interface{}{
							"ProductName":     "test-pro",
							"SystemID":        "objectFakeID",
							"FirmwareVersion": "2.0",
						},
					},
					"RegistryEntries": map[string]interface{}{
						"Attributes": []interface{}{
							map[string]interface{}{
								"Value": "263",
							}},
					},
				},
				bmcObj: &infraiov1.Bmc{
					ObjectMeta: metav1.ObjectMeta{Name: "10.10.10.10", Namespace: "bmc-op"},
					Status:     infraiov1.BmcStatus{BmcSystemID: "objectFakeID"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.CheckAndCreateBiosSchemaObject(tt.args.ctx, tt.args.attributeResp, tt.args.bmcObj); got != tt.want {
				t.Errorf("CommonReconciler.CheckAndCreateBiosSchemaObject() = %v, want %v", got, tt.want)
			}
		})
	}
}