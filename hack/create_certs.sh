#!/bin/sh
# taken from here: https://itnext.io/practical-guide-to-securing-grpc-connections-with-go-and-tls-part-1-f63058e9d6d1
openssl genrsa -out keys/ca.key 4096
openssl req -new -x509 -key keys/ca.key -sha256 -subj "/C=US/ST=CA/O=haiku.io" -days 365 -out keys/ca.cert
openssl genrsa -out keys/service.key 4096
openssl req -new -key keys/service.key -out keys/service.csr -config hack/certificate.conf
openssl x509 -req -in keys/service.csr -CA keys/ca.cert -CAkey keys/ca.key -CAcreateserial -out keys/service.pem -days 365 -sha256 -extfile hack/certificate.conf -extensions req_ext
