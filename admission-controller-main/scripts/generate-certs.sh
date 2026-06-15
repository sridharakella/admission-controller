#!/bin/bash

# Script to generate self-signed certificates for the admission controller webhook
# This is an alternative to using cert-manager

set -e

NAMESPACE="admission-controller"
SERVICE_NAME="admission-controller"
SECRET_NAME="admission-controller-tls"
CERT_DIR="$(pwd)/certs"

echo "Generating TLS certificates for admission controller..."

# Create certs directory
mkdir -p "${CERT_DIR}"

# Generate CA private key
openssl genrsa -out "${CERT_DIR}/ca.key" 2048

# Generate CA certificate
openssl req -x509 -new -nodes -key "${CERT_DIR}/ca.key" \
  -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" \
  -days 3650 \
  -out "${CERT_DIR}/ca.crt"

# Generate server private key
openssl genrsa -out "${CERT_DIR}/tls.key" 2048

# Create certificate signing request (CSR)
cat > "${CERT_DIR}/csr.conf" <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE_NAME}
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}
DNS.3 = ${SERVICE_NAME}.${NAMESPACE}.svc
DNS.4 = ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local
EOF

# Generate CSR
openssl req -new -key "${CERT_DIR}/tls.key" \
  -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" \
  -out "${CERT_DIR}/tls.csr" \
  -config "${CERT_DIR}/csr.conf"

# Generate server certificate
openssl x509 -req -in "${CERT_DIR}/tls.csr" \
  -CA "${CERT_DIR}/ca.crt" \
  -CAkey "${CERT_DIR}/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/tls.crt" \
  -days 3650 \
  -extensions v3_req \
  -extfile "${CERT_DIR}/csr.conf"

echo "Certificates generated successfully in ${CERT_DIR}/"
echo ""
echo "To create the Kubernetes secret, run:"
echo "  kubectl create secret tls ${SECRET_NAME} \\"
echo "    --cert=${CERT_DIR}/tls.crt \\"
echo "    --key=${CERT_DIR}/tls.key \\"
echo "    -n ${NAMESPACE}"
echo ""
echo "To get the CA bundle for the webhook configuration, run:"
echo "  cat ${CERT_DIR}/ca.crt | base64 | tr -d '\\n'"
echo ""

# Optionally create the secret automatically
read -p "Create the Kubernetes secret now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
  kubectl create secret tls ${SECRET_NAME} \
    --cert="${CERT_DIR}/tls.crt" \
    --key="${CERT_DIR}/tls.key" \
    -n ${NAMESPACE} \
    --dry-run=client -o yaml | kubectl apply -f -
  echo "Secret created successfully!"

  echo ""
  echo "CA Bundle (add this to your webhook configuration if not using cert-manager):"
  cat "${CERT_DIR}/ca.crt" | base64 | tr -d '\n'
  echo ""
fi
