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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"

	l "github.com/ODIM-Project/BMCOperator/logs"
)

// Encryption with OAEP padding
func EncryptWithPublicKey(ctx context.Context, secretMessage string,
	key rsa.PublicKey) string {

	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha512.New(),
		rng, &key, []byte(secretMessage), nil)

	if err != nil {
		l.LogWithFields(ctx).Errorf("Unable to encrypt password: %s", err.Error())
		return ""
	}

	return base64.StdEncoding.EncodeToString(ciphertext)
}

// Decryption
func DecryptWithPrivateKey(ctx context.Context, cipherText string,
	privKey rsa.PrivateKey, doBase64Decode bool) string {
	ct := []byte(cipherText)
	if doBase64Decode {
		//Decode the Cipher text
		var err error
		ct, err = base64.StdEncoding.DecodeString(cipherText)
		if err != nil {
			l.LogWithFields(ctx).Errorf("Unable to decode password: %s", err.Error())
			return ""
		}
	}
	rng := rand.Reader
	secrettext, err := rsa.DecryptOAEP(sha512.New(),
		rng, &privKey, ct, nil)

	if err != nil {
		l.LogWithFields(ctx).Errorf("Unable to decrypt password: %s", err.Error())
		return ""
	}
	return string(secrettext)
}
