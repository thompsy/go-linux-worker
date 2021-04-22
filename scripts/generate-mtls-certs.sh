#!/bin/sh

# This script generates the following certificates:
# * a trusted CA cert
# * a server key and certificate signed by the trusted CA
# * two client keys and certificates signed by the trusted CA
# * an untrusted CA cert
# * a client key and certificate signed by the untrusted CA

# todo:
# * what about the config files that I'm not using any more?

OUTPUT_DIR=./certs
mkdir -p ${OUTPUT_DIR}



# Generate the CA cert
# nodes skips the password.
#todo update other keys below with rsa:4096. We should be able to trim the conf files too.
#todo see: https://github.com/gravitational/teleport/blob/740d184d1cfc69ae2e96c50ee738b13884fb232b/examples/etcd/certs/Makefile
#todo: see https://www.keylength.com/en/compare/
openssl req \
  -newkey rsa:4096 \
  -x509 \
  -nodes \
  -days 365 \
  -subj '/CN=worker-api-ca' \
  -keyout ${OUTPUT_DIR}/ca.key \
  -out ${OUTPUT_DIR}/ca.crt



# Generate server key
openssl genrsa -out ${OUTPUT_DIR}/server.key 2048
# Create the CSR for the server
openssl req \
  -new \
  -key ${OUTPUT_DIR}/server.key \
  -out ${OUTPUT_DIR}/server.csr \
  -config scripts/server-certificate.conf
# Create the signed certificate for the server
openssl x509 \
  -req \
  -in ${OUTPUT_DIR}/server.csr \
  -CA ${OUTPUT_DIR}/ca.crt \
  -CAkey ${OUTPUT_DIR}/ca.key \
  -CAcreateserial \
  -days 365 \
  -out ${OUTPUT_DIR}/server.crt \
  -sha256 \
  -extfile scripts/server-certificate.conf \
  -extensions req_ext



# Generate the key for client A
openssl genrsa \
  -out ${OUTPUT_DIR}/client_a.key 2048

# Create the CSR for client A
openssl req \
  -new \
  -key ${OUTPUT_DIR}/client_a.key \
  -out ${OUTPUT_DIR}/client_a.csr \
  -subj "/C=US/ST=CA/O=Acme, Inc./CN=client_a@example.com"
#  -config scripts/client-certificate.conf
# Create the signed certificate for client A
openssl x509 \
  -req \
  -in ${OUTPUT_DIR}/client_a.csr \
  -CA ${OUTPUT_DIR}/ca.crt \
  -CAkey ${OUTPUT_DIR}/ca.key \
  -CAcreateserial \
  -days 365 \
  -out ${OUTPUT_DIR}/client_a.crt



# Generate the key for client B
openssl genrsa \
	-out ${OUTPUT_DIR}/client_b.key 2048
# Create the CSR for client B
openssl req \
  -new \
  -key ${OUTPUT_DIR}/client_b.key \
  -out ${OUTPUT_DIR}/client_b.csr \
  -subj "/C=US/ST=CA/O=Acme, Inc./CN=client_b@example.com"
# Create the signed certificate for client B
openssl x509 \
  -req \
  -in ${OUTPUT_DIR}/client_b.csr \
  -CA ${OUTPUT_DIR}/ca.crt \
  -CAkey ${OUTPUT_DIR}/ca.key \
  -CAcreateserial \
  -days 365 \
  -out ${OUTPUT_DIR}/client_b.crt



# Generate the key for admin client
openssl genrsa \
	-out ${OUTPUT_DIR}/client_admin.key 2048
# Create the CSR for client B
openssl req \
  -new \
  -key ${OUTPUT_DIR}/client_admin.key \
  -out ${OUTPUT_DIR}/client_admin.csr \
  -subj "/C=US/ST=CA/O=Acme, Inc./CN=admin@example.com"
# Create the signed certificate for admin client
openssl x509 \
  -req \
  -in ${OUTPUT_DIR}/client_admin.csr \
  -CA ${OUTPUT_DIR}/ca.crt \
  -CAkey ${OUTPUT_DIR}/ca.key \
  -CAcreateserial \
  -days 365 \
  -out ${OUTPUT_DIR}/client_admin.crt



# Generate CA cert that is unrecognised by the server for testing
# Generate the CA cert
openssl req \
  -new \
  -x509 \
  -nodes \
  -days 365 \
  -subj '/CN=untrusted-ca' \
  -keyout ${OUTPUT_DIR}/untrusted_ca.key \
  -out ${OUTPUT_DIR}/untrusted_ca.crt



# Generate the untrusted client key
openssl genrsa \
	-out ${OUTPUT_DIR}/untrusted_client.key 2048
# Create the CSR for the untrusted client
openssl req \
  -new \
  -key ${OUTPUT_DIR}/untrusted_client.key \
  -out ${OUTPUT_DIR}/untrusted_client.csr \
  -subj "/C=US/ST=CA/O=Acme, Inc./CN=admin@example.com"
# Create the signed certificate for the untrusted client
openssl x509 \
  -req \
  -in ${OUTPUT_DIR}/untrusted_client.csr \
  -CA ${OUTPUT_DIR}/untrusted_ca.crt \
  -CAkey ${OUTPUT_DIR}/untrusted_ca.key \
  -CAcreateserial \
  -days 365 \
  -out ${OUTPUT_DIR}/untrusted_client.crt
