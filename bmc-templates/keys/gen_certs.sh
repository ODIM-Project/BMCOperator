# (C) Copyright [2023] Hewlett Packard Enterprise Development LP
# 
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
#  under the License.

#!/bin/bash

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
#generate server csr
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
#sign and gen server certificate
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

rootCA=$(cat rootCA.crt | base64)
clientCrt=$(cat ${FileName}.crt | base64)
clientKey=$(cat ${FileName}.key | base64)

yq eval ".data.rootCACert = \"$rootCA\"" secret.yaml > secret1.yaml
yq eval ".data.eventClientCert = \"$clientCrt\"" secret1.yaml > secret2.yaml
yq eval ".data.eventClientKey = \"$clientKey\"" secret2.yaml > secret3.yaml
rm secret.yaml secret1.yaml secret2.yaml
mv secret3.yaml secret.yaml

openssl genrsa -out bmc.private 4096
openssl rsa -in bmc.private -out bmc.public -pubout -outform PEM
publicKey=$(cat bmc.public | base64)
privateKey=$(cat bmc.private | base64)
yq eval ".data.privateKey = \"$privateKey\"" secret.yaml > secret1.yaml
yq eval ".data.publicKey = \"$publicKey\"" secret1.yaml > secret2.yaml
rm secret.yaml secret1.yaml
mv secret2.yaml secret.yaml