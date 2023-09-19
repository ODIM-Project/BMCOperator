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
	"crypto/rsa"
	"reflect"
	"regexp"
	"sort"
	"strings"

	infraiov1 "github.com/ODIM-Project/BMCOperator/api/v1"
)

var (
	// PrivateKey holds the value for PrivateKey
	PrivateKey *rsa.PrivateKey
	// PublicKey holds the value for PublicKey
	PublicKey *rsa.PublicKey
	// RootCA contains the RootCA value
	RootCA []byte
)

// ConvertToArray is used to convert interface to array of string
func ConvertToArray(data []interface{}) []string {
	aString := make([]string, len(data))
	for i, v := range data {
		aString[i] = v.(string)
	}
	return aString
}

// CompareArray is used to compare two arrays
func CompareArray(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	arr1Copy := make([]string, len(arr1))
	arr2Copy := make([]string, len(arr2))

	copy(arr1Copy, arr1)
	copy(arr2Copy, arr2)

	sort.Strings(arr1Copy)
	sort.Strings(arr2Copy)

	return reflect.DeepEqual(arr1Copy, arr2Copy)
}

// ContainsValue checks if array contains the string value
func ContainsValue(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

// RemoveSpecialChar used to remove special char from ID to get bios object name
func RemoveSpecialChar(str string) string {
	re, _ := regexp.Compile(`[^a-zA-Z0-9 ]+`)
	str = re.ReplaceAllString(str, " ")
	biosName := strings.ReplaceAll(str, " ", "")
	biosName = strings.ToLower(biosName)
	return biosName
}

// CaseInsensitiveContains used to check if vendor is available
func CaseInsensitiveContains(s string) (string, bool) {
	for _, substr := range infraiov1.Vendor {
		s, substr = strings.ToUpper(s), strings.ToUpper(substr)
		if strings.Contains(s, substr) {
			return substr, true
		}
	}
	return "", false
}

// CompareMaps function compares values of two maps and return true if equal else returns false
func CompareMaps(m1, m2 map[string]string) (bool, map[string]string) {
	distinctMap := make(map[string]string)
	if reflect.DeepEqual(m1, m2) {
		return true, nil
	}
	for k, v := range m1 {
		if m2[k] != v {
			distinctMap[k] = v
		}
	}

	return false, distinctMap
}

// ConvertInterfaceToStringArray will convert and return an array of interface into array of string
func ConvertInterfaceToStringArray(items []interface{}) []string {
	var values []string
	for _, item := range items {
		values = append(values, item.(string))
	}
	return values
}
