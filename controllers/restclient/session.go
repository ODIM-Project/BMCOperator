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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func (rc *RestClient) getNewTokenAndUpdateOdimClient() error {
	session := getSessionObject(rc.username, rc.password, rc.host, rc.port)
	token, err := session.getXAuthToken(rc.rootCA)
	if err != nil {
		return fmt.Errorf("Error: Generating new token for ODIM" + err.Error())
	}
	headerDetails := rc.reqHeaderAuth.getHeaderDetails()
	headerDetails["X-Auth-Token"] = token
	// rc.reqHeaderAuth.(*requestDetails).Auth["X-Auth-Token"] = token
	// odimRcs.(*RestClient).headerAuthDetails.(*headerAuth).Auth["X-Auth-Token"] = token
	return nil
}

func (token createTokenDetails) getXAuthToken(rootCA []byte) (string, error) {
	sessionURL := fmt.Sprintf("https://%s:%s/redfish/v1/SessionService/Sessions", token.host, token.port)
	creds := map[string]string{"UserName": token.username, "Password": token.password}
	body, err := json.Marshal(creds)
	if err != nil {
		return "", err
	}
	conn := getConn(rootCA)
	// Create a HTTP post request
	req, err := http.NewRequest("POST", sessionURL, bytes.NewBuffer(body))
	if err != nil {
		return "Unable to create request for session!", err
	}
	response, err := conn.Do(req)
	if err != nil {
		return "Error in sending Request", err
	}
	if response.StatusCode == http.StatusCreated {
		return response.Header["X-Auth-Token"][0], nil
	} else if response.StatusCode == http.StatusUnauthorized {
		return "", errors.New("Username/Password is wrong for session creation!, Check again")
	}
	return response.Status, nil
}

// RecallWithNewToken calls the same Rest call with new token
func (rc *RestClient) RecallWithNewToken(url, reason, method string, body interface{}) (*http.Response, error) {
	err := rc.getNewTokenAndUpdateOdimClient()
	if err != nil {
		return nil, err
	}
	resp, err := rc.reqHeaderAuth.sendRequest(method, reason, url, body, rc.rootCA)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func getSessionObject(username, password, host, port string) sessionInterface {
	return createTokenDetails{username: username, password: password, host: host, port: port}
}
