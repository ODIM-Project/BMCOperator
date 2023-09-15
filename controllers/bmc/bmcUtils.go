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
	"crypto/rsa"
	"fmt"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	common "github.com/ODIM-Project/BMCOperator/controllers/common"
	restclient "github.com/ODIM-Project/BMCOperator/controllers/restclient"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// bios "github.com/ODIM-Project/BMCOperator/controllers/bios"
	// boot "github.com/ODIM-Project/BMCOperator/controllers/boot"
	l "github.com/ODIM-Project/BMCOperator/logs"
	corev1 "k8s.io/api/core/v1"
	types "k8s.io/apimachinery/pkg/types"
)

const (
	bmcFinalizer = "infra.io.bmc/finalizer"
	biosGroup    = "infra.io.odimra"
	biosKind     = "BiosSetting"
	biosVersion  = "v1"
	bootGroup    = "infra.io.odimra"
	bootKind     = "BootOrderSetting"
	bootVersion  = "v1"
	volFinalizer = "infra.io.volume/finalizer"
)

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

// modifyPassword struct is used for creating modify password request
type modifyPassword struct {
	Password string
}

type BmcInterface interface {
	AddBmc(body []byte, namespaceName types.NamespacedName, sysID string) bool
	PrepareAddBmcPayload(connectionMeth string) ([]byte, error)
	UpdateBmcLabels()
	UpdateDriveDetails() map[string]infraiov1.ArrayControllers
	// GetSystemDetails() map[string]interface{}
	GetManagerDetails() map[string]interface{}
	UnregisterBmc() bool
	LoggingBmcDeletionActivity(isDeleted error, objectName string)
	RemoveBmcFinalizerAndDeleteDependencies()
	GetAllowableResetValues() []string
	ValidateResetData(powerState string, allowableValues []string) bool
	ResetSystem(isBiosUpdation bool, updateBmcDependents bool) bool
	UpdateOdimWithNewPassword() (bool, error)
	UpdateBmcWithNewPassword() (bool, error)
	EncryptPassword(publicKey *rsa.PublicKey)
	GetConnectionMethod(odimObj *infraiov1.Odim) string
	UpdateBmcObject(bmcObj *infraiov1.Bmc)
	DeleteBMCObject()
}

type bmcUtils struct {
	ctx             context.Context
	bmcObj          *infraiov1.Bmc
	namespace       string
	commonRec       utils.ReconcilerInterface
	bmcRestClient   restclient.RestClientInterface
	commonUtil      common.CommonInterface
	updateBMCObject bool
}

// ----------- Get Keys for BMC encryption--------------
// getPemKeys will read the pem files and parse the private and public key
func GetPemKeys(ctx context.Context, pubKey, privKey string) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := utils.ParseRsaPrivateKeyFromPemStr(privKey)
	if err != nil {
		l.LogWithFields(ctx).Error("Error parsing private key: " + err.Error())
		return nil, nil
	}
	publicKey, err := utils.ParseRsaPublicKeyFromPemStr(pubKey)
	if err != nil {
		l.LogWithFields(ctx).Error("Error parsing public key: " + err.Error())
		return nil, nil
	}
	return privateKey, publicKey
}

// getKeysSecret retreives the keys from secret
func GetEncryptedPemKeysFromSecret(ctx context.Context, secret *corev1.Secret, secretName, namespace string, r client.Client) (string, string) {
	ns := types.NamespacedName{Namespace: namespace, Name: secretName}
	err := r.Get(context.TODO(), ns, secret)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching %s secret:", secretName), err.Error())
	}
	publicKey := string(secret.Data["publicKey"])
	privateKey := string(secret.Data["privateKey"])
	return publicKey, privateKey
}
