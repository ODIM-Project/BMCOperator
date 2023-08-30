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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"

	config "github.com/ODIM-Project/BMCOperator/controllers/config"
	l "github.com/ODIM-Project/BMCOperator/logs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// GetHTTPServerObj is for obtaining a server instance to start the event listener using iris helper
func GetHTTPServerObj(ecr *EventsClientReconciler) (*http.Server, error) {
	tlsConfig := &tls.Config{}
	if err := LoadCertificates(tlsConfig, ecr); err != nil {
		return nil, err
	}
	tlsConfig.MinVersion = tls.VersionTLS12
	tlsConfig.MaxVersion = tls.VersionTLS12

	return &http.Server{
		Addr:      net.JoinHostPort("", config.Data.EventClientPort),
		TLSConfig: tlsConfig,
	}, nil
}

// LoadCertificates is for including passed certificates in tls.Config
func LoadCertificates(tlsConfig *tls.Config, ecr *EventsClientReconciler) error {

	rootCA, eventClientCert, eventClientKey := GetSignedCertificatesForEventClient(context.TODO(), ecr)

	cert, err := tls.X509KeyPair(eventClientCert, eventClientKey)
	if err != nil {
		return fmt.Errorf("error: failed to load key pair: %v", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	capool := x509.NewCertPool()
	if !capool.AppendCertsFromPEM(rootCA) {
		return fmt.Errorf("error: failed to load CA certificate")
	}

	tlsConfig.RootCAs = capool
	tlsConfig.ClientCAs = capool
	return nil
}

// GetSignedCertificatesForEventClient retrieves the certificates from secret
func GetSignedCertificatesForEventClient(ctx context.Context, ecr *EventsClientReconciler) (rootCA, eventClientCert, eventClientKey []byte) {
	secret := &corev1.Secret{}
	ns := types.NamespacedName{Namespace: config.Data.Namespace, Name: config.Data.SecretName}
	err := ecr.Get(context.TODO(), ns, secret)
	if err != nil {
		l.LogWithFields(ctx).Error(fmt.Sprintf("Error fetching %s secret", config.Data.SecretName), err.Error())
		return
	}
	return secret.Data["rootCACert"],
		secret.Data["eventClientCert"],
		secret.Data["eventClientKey"]
}
