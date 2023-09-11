#!/bin/bash

#(C) Copyright [2023] Hewlett Packard Enterprise Development LP
#
#Licensed under the Apache License, Version 2.0 (the "License"); you may
#not use this file except in compliance with the License. You may obtain
#a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
#WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
#License for the specific language governing permissions and limitations
#under the License.

FileName=client
clientIP=$1
if [ -z "$1" ]
then
    echo "Please provide clientIP"
    exit -1
else
    echo "clientIP is $1"
fi

#generate server private key
openssl genrsa -out ${FileName}.key 4096
echo
echo

openssl req -new -sha512 -key ${FileName}.key -subj "/C=US/ST=CA/O=ODIMRA/CN=Event Client Cert" -config <(cat <<EOF
[ req ]
prompt = no
distinguished_name = subject
req_extensions    = req_ext
[ subject ]
commonName = Event Client Cert
[ req_ext ]
extendedKeyUsage=serverAuth, clientAuth
basicConstraints=critical,CA:false
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
EOF
) -out ${FileName}.csr
echo
echo

openssl x509 -req -extensions server_crt -extfile <( cat <<EOF
[server_crt]
basicConstraints=CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage=serverAuth, clientAuth
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
subjectAltName = @alternate_names
[ alternate_names ]
IP.0 = $clientIP
EOF
) -in  ${FileName}.csr -CA  rootCA.crt -CAkey  rootCA.key  -CAcreateserial -out ${FileName}.crt -days 500 -sha512


openssl genrsa -out bmc.private 4096
openssl rsa -in bmc.private -out bmc.public -pubout -outform PEM

rootCA=$(cat rootCA.crt)
clientCrt=$(cat ${FileName}.crt)
clientKey=$(cat ${FileName}.key)

# Append values to values.yaml
yq eval ".data.rootCACert = \"$rootCA\"" values.yaml > values1.yaml
yq eval ".data.eventClientCert = \"$clientCrt\"" values1.yaml > values2.yaml
yq eval ".data.eventClientKey = \"$clientKey\"" values2.yaml > values3.yaml
rm values.yaml values1.yaml values2.yaml
mv values3.yaml values.yaml

publicKey=$(cat bmc.public)
privateKey=$(cat bmc.private)

# Append values to values.yaml
yq eval ".data.privateKey = \"$privateKey\"" values.yaml > values1.yaml
yq eval ".data.publicKey = \"$publicKey\"" values1.yaml > values2.yaml
rm values.yaml values1.yaml
mv values2.yaml values.yaml