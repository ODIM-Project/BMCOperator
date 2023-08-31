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
	"net/http"
)

// --------------REST CLIENT--------------------------
type requestInterface interface {
	sendRequest(string, string, string, interface{}, []byte) (*http.Response, error)
	getAuthType() string
	getHeaderDetails() map[string]string
}

type RestClientInterface interface {
	Post(string, string, interface{}) (*http.Response, error)
	Get(string, string) (map[string]interface{}, int, error)
	Patch(string, string, interface{}) (*http.Response, error)
	Delete(string, string) (*http.Response, error)
	Put(string, string, interface{}) (*http.Response, error)
	RecallWithNewToken(url, reason, method string, body interface{}) (*http.Response, error)
}

type RestClient struct {
	reqHeaderAuth requestInterface
	host          string
	port          string
	username      string
	password      string
	rootCA        []byte
}

type requestDetails struct {
	Headers  map[string]string
	AuthType string
}

//--------------SESSION----------------

type sessionInterface interface {
	getXAuthToken([]byte) (string, error)
}

type createTokenDetails struct {
	username string
	password string
	host     string
	port     string
}

//------------ HTTP CONNECTION----------------

// HTTPClient interface
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}
