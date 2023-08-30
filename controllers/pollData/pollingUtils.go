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

// Package controllers ...
package controllers

import (
	"context"
	"fmt"
	"os"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	"github.com/ODIM-Project/BMCOperator/config/constants"
	bmc "github.com/ODIM-Project/BMCOperator/controllers/bmc"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	volume "github.com/ODIM-Project/BMCOperator/controllers/volume"
	l "github.com/ODIM-Project/BMCOperator/logs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	podName = os.Getenv("POD_NAME")
)

// PollingReconciler defines a struct of object and interfaces
type PollingReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	odimObj        *infraiov1.Odim
	bmcObject      *infraiov1.Bmc
	ctx            context.Context
	commonRec      utils.ReconcilerInterface
	pollRestClient restclient.RestClientInterface
	commonUtil     common.CommonInterface
	volUtil        volume.VolumeInterface
	bmcUtil        bmc.BmcInterface
	namespace      string
}

// links struct is used in add bmc body
type links struct {
	ConnectionMethod map[string]string
}

// addBmcPayloadBody struct is used for creating add bmc payload
type addBmcPayloadBody struct {
	HostName string
	UserName string
	Password string
	Links    links
}

// getKeysSecret retreives the keys from secret
func (r *PollingReconciler) getEncryptedPemKeysFromSecret(ctx context.Context, secret *corev1.Secret, secretName, namespace string) (string, string) {
	ns := types.NamespacedName{Namespace: namespace, Name: secretName}
	err := r.Get(context.TODO(), ns, secret)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching %s secret", secretName), err.Error())
	}
	publicKey := string(secret.Data["publicKey"])
	privateKey := string(secret.Data["privateKey"])
	return publicKey, privateKey
}

func GetPollingReconciler(ctx context.Context, client client.Client, Scheme *runtime.Scheme) (*PollingReconciler, error) {
	r := &PollingReconciler{Client: client, Scheme: Scheme}

	if r.odimObj == nil {
		r.commonRec = utils.GetCommonReconciler(r.Client, r.Scheme)
		r.odimObj = r.commonRec.GetOdimObject(ctx, constants.MetadataName, "odim", r.namespace)
		if r.odimObj == nil {
			return nil, fmt.Errorf("odim Object is not present")
		}
	}
	return r, nil
}
