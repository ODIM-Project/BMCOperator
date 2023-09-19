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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
)

var (
	// MuxLock is used for avoiding race conditions
	MuxLock = &sync.Mutex{}
)

// ExportRsaPrivateKeyAsPemStr returns private key as pem string
func ExportRsaPrivateKeyAsPemStr(privkey *rsa.PrivateKey) string {
	privkeyBytes, err := x509.MarshalPKCS8PrivateKey(privkey)
	if err != nil {
		privkeyBytes = x509.MarshalPKCS1PrivateKey(privkey)
	}
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	return string(privkeyPem)
}

// ExportRsaPublicKeyAsPemStr returns public key as pem string
func ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) (string, error) {
	pubkeyBytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", err
	}
	pubkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkeyBytes,
		},
	)

	return string(pubkeyPem), nil
}

// ParseRsaPublicKeyFromPemStr returns the public key from string
func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("Key type is not RSA")
}

// ParseRsaPrivateKeyFromPemStr returns the private key from string
func ParseRsaPrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
	MuxLock.Lock()
	defer MuxLock.Unlock()
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var priv *rsa.PrivateKey
	if enc {
		bb, err := x509.DecryptPEMBlock(block, nil)
		if err != nil {
			return nil, fmt.Errorf("error while trying to decrypt pem block: %v", err)
		}
		b = bb
	}
	priv, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		pkcs8Key, err := x509.ParsePKCS8PrivateKey(b)
		if err != nil {
			return nil, fmt.Errorf("error while parsing private key %v", err)
		}
		priv = pkcs8Key.(*rsa.PrivateKey)
	}

	return priv, nil
}
