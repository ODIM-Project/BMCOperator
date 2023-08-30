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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateBmcObject(t *testing.T) {
	client := &MockClient{}
	reconciler := PollingReconciler{
		Client:    client,
		Scheme:    nil,
		odimObj:   nil,
		bmcObject: nil,
		commonRec: nil,
	}
	bmcObj := reconciler.createBmcObject(client, "Bmc", "10.1.1.1", "1", "admin", "Compute:BasicAuth:ILO_v2.0.0")
	assert.Equal(t, "infra.io.odimra/v1, Kind=Bmc", bmcObj.GroupVersionKind().String())
	assert.Equal(t, "10.1.1.1", bmcObj.GetName())
	assert.Equal(t, "", bmcObj.GetNamespace())
	assert.Equal(t, "Compute:BasicAuth:ILO_v2.0.0", bmcObj.Spec.BmcDetails.ConnMethVariant)
	assert.Equal(t, "1", bmcObj.Status.BmcSystemID)
	assert.Equal(t, "10.1.1.1", bmcObj.Spec.BmcDetails.Address)
	assert.Equal(t, "admin", bmcObj.Spec.Credentials.Username)
	assert.Equal(t, map[string]string{"systemId": "1", "name": "10.1.1.1"}, bmcObj.Labels)
	annotations := bmcObj.GetAnnotations()
	assert.NotNil(t, annotations)
}
