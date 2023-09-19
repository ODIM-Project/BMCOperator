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

// Package controllers ...
package controllers

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
	utils "github.com/ODIM-Project/BMCOperator/controllers/utils"
	l "github.com/ODIM-Project/BMCOperator/logs"
	corev1 "k8s.io/api/core/v1"
)

var (
	httpConn HTTPClientInterface
	err      error
)

// -------------------------------NEW CLIENT CALLS---------------------------------

// NewRestClient creates rest client for odim
func NewRestClient(ctx context.Context, odimObj *infraiov1.Odim, commonRec *utils.CommonReconciler, rootdir string) (RestClientInterface, error) {
	//getting secrets
	secret := corev1.Secret{}
	var secretName string
	for field, val := range odimObj.Annotations {
		if strings.Contains(field, "auth") {
			secretName = val
		}
	}
	username, password, authType := commonRec.GetObjectSecret(ctx, &secret, secretName, rootdir)
	//get password in base64 form
	pass := utils.DecryptWithPrivateKey(ctx, password, *utils.PrivateKey, false) //doBase64Decode = false, because odim password fetched from secret is already decoded
	l.LogWithFields(ctx).Info("Fetching Odim address")
	//get host,port
	host, port, err := utils.GetHostPort(ctx, odimObj.Spec.URL)
	if err != nil {
		l.LogWithFields(ctx).Error("Error fetching host/port of ODIM" + err.Error())
		return nil, err
	}
	rootCA := utils.RootCA
	//Create rc object
	restClient, err := getNewRestClient(ctx, username, pass, authType, host, port, rootCA)
	if err != nil {
		l.LogWithFields(ctx).Error("Error in creating odim rest client." + err.Error())
		return nil, err
	}
	return restClient, nil
}

// getNewRestClient returns Rest client
func getNewRestClient(ctx context.Context, username, password, authType, host, port string, rootCA []byte) (RestClientInterface, error) {
	l.LogWithFields(ctx).Info("Creating new rest client for ODIM")
	userPass := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", strings.TrimSpace(username), strings.TrimSpace(password))))
	var auth map[string]string
	if strings.Contains(authType, "Basic") { // authType = BasicAuth
		auth = map[string]string{"Authorization": fmt.Sprintf("Basic %s", userPass)}
	} else if strings.Contains(authType, "Session") { // authType = RedfishSessionAuth
		session := getSessionObject(username, password, host, port)
		xAuthToken, err := session.getXAuthToken(rootCA)
		if err != nil {
			return nil, err
		}
		auth = map[string]string{"X-Auth-Token": xAuthToken}
	}
	headers := map[string]string{"Content-Type": "application/json",
		"Accept": "application/json"}
	//adding authentication details to headers
	for key, val := range auth {
		headers[key] = val
	}
	return &RestClient{
		reqHeaderAuth: NewHeaderAuth(headers, authType),
		host:          host,
		port:          port,
		username:      username,
		password:      password,
		rootCA:        rootCA,
	}, nil
}

// NewHeaderAuth returns requestInterface struct
func NewHeaderAuth(headers map[string]string, authType string) requestInterface {
	return &requestDetails{
		Headers:  headers,
		AuthType: authType,
	}
}

// -----------------------------REST CALLS----------------------------------

// Post rest call
func (rc *RestClient) Post(uri, reason string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("https://%s:%s%s", rc.host, rc.port, uri)
	resp, err := rc.reqHeaderAuth.sendRequest(http.MethodPost, reason, url, body, rc.rootCA)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusUnauthorized && strings.Contains(rc.reqHeaderAuth.getAuthType(), "Session") {
		return rc.RecallWithNewToken(url, http.MethodPost, reason, body)
	} else if resp.StatusCode == http.StatusUnauthorized {
		return resp, nil
	}
	return resp, nil
}

// Patch rest call
func (rc *RestClient) Patch(uri, reason string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("https://%s:%s%s", rc.host, rc.port, uri)
	resp, err := rc.reqHeaderAuth.sendRequest(http.MethodPatch, reason, url, body, rc.rootCA)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusUnauthorized && strings.Contains(rc.reqHeaderAuth.getAuthType(), "Session") { // TODO: what if password given is wrong // handle that case
		return rc.RecallWithNewToken(url, http.MethodPatch, reason, body)
	} else if resp.StatusCode == http.StatusUnauthorized {
		return resp, nil
	}
	return resp, nil
}

// Put rest call
func (rc *RestClient) Put(uri, reason string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("https://%s:%s%s", rc.host, rc.port, uri)
	resp, err := rc.reqHeaderAuth.sendRequest(http.MethodPut, reason, url, body, rc.rootCA)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusUnauthorized && strings.Contains(rc.reqHeaderAuth.getAuthType(), "Session") {
		return rc.RecallWithNewToken(url, http.MethodPut, reason, body)
	} else if resp.StatusCode == http.StatusUnauthorized {
		return resp, nil
	}
	return resp, nil
}

// Delete rest call
func (rc *RestClient) Delete(uri, reason string) (*http.Response, error) {
	url := fmt.Sprintf("https://%s:%s%s", rc.host, rc.port, uri)
	resp, err := rc.reqHeaderAuth.sendRequest(http.MethodDelete, reason, url, "", rc.rootCA)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusUnauthorized {
		return rc.RecallWithNewToken(url, http.MethodDelete, reason, "")
	}
	return resp, nil
}

// Get rest call
func (rc *RestClient) Get(uri, reason string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("https://%s:%s%s", rc.host, rc.port, uri)
	var data map[string]interface{}
	resp, err := rc.reqHeaderAuth.sendRequest(http.MethodGet, reason, url, "", rc.rootCA)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode == 308 {
		data, statusCode, err := rc.Get((uri + "/"), reason)
		if err != nil {
			return nil, statusCode, err
		}
		return data, resp.StatusCode, err
	} else if resp.StatusCode == http.StatusUnauthorized && strings.Contains(rc.reqHeaderAuth.getAuthType(), "Session") {
		resp, err = rc.RecallWithNewToken(url, http.MethodGet, reason, "")
		if err != nil {
			return nil, 0, err
		}
	} // handle 401
	resbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(resbody) == 0 { // for DELETE query's taskmon (doesnt contain Response Body)
		return nil, resp.StatusCode, nil
	}
	err = json.Unmarshal([]byte(resbody), &data)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

// -----------------------------------REQUEST CALLS------------------------------------
// sendRequest sends the request
func (request *requestDetails) sendRequest(method, reason, uri string, body interface{}, rootCA []byte) (*http.Response, error) {
	//creating new req
	var req *http.Request
	//if method = GET/DELETE , body is empty
	if method == "GET" || method == "DELETE" {
		req, err = http.NewRequest(method, uri, nil)
		if err != nil {
			errMsg := fmt.Errorf("Error: Creating HTTP request %s on %s to query ODIM: %s", method, uri, err.Error())
			return nil, errMsg
		}
	} else {
		req, err = http.NewRequest(method, uri, bytes.NewBuffer(body.([]byte)))
		if err != nil {
			errMsg := fmt.Errorf("Error: Creating HTTP request %s on %s to query ODIM: %s", method, uri, err.Error())
			return nil, errMsg
		}
	}
	// setting header
	for header, val := range request.Headers {
		req.Header.Set(header, val)
	}
	if httpConn == nil {
		httpConn = getConn(rootCA)
	}
	res, err := httpConn.Do(req)
	if err != nil {
		errMsg := fmt.Errorf("Error: Sending HTTP request %s on %s to query ODIM: %s", method, uri, err.Error())
		return nil, errMsg
	}
	return res, nil
}

// getAuthType returns authentication type
func (request *requestDetails) getAuthType() string {
	return request.AuthType
}

func (request *requestDetails) getHeaderDetails() map[string]string {
	return request.Headers
}

// ------------------------HTTP CONNECTION--------------------------
// genConn establishes connection
func getConn(rootCA []byte) HTTPClientInterface {
	capool := x509.NewCertPool()
	capool.AppendCertsFromPEM(rootCA)

	connection := &http.Client{
		Transport: &http.Transport{
			DialTLS: func(network, addr string) (net.Conn, error) {
				conn, err := tls.Dial(network, addr, &tls.Config{
					RootCAs: capool,
				})
				return conn, err
			},
		},
	}
	return connection
}
