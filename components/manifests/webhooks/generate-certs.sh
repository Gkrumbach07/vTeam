#!/bin/bash

set -e

NAMESPACE=ambient-system
SERVICE=ambient-webhook-service
SECRET=ambient-webhook-certs

# Create a temporary directory for certificates
TMPDIR=$(mktemp -d)
echo "Working in temporary directory: $TMPDIR"

# Generate CA private key
openssl genrsa -out $TMPDIR/ca.key 2048

# Generate CA certificate
openssl req -new -x509 -days 365 -key $TMPDIR/ca.key \
    -subj "/C=US/ST=CA/L=SF/O=AmbientAI/CN=webhook-ca" \
    -out $TMPDIR/ca.crt

# Generate server private key
openssl genrsa -out $TMPDIR/tls.key 2048

# Generate certificate signing request
openssl req -newkey rsa:2048 -nodes -keyout $TMPDIR/tls.key \
    -subj "/C=US/ST=CA/L=SF/O=AmbientAI/CN=$SERVICE.$NAMESPACE.svc" \
    -out $TMPDIR/server.csr

# Generate server certificate
openssl x509 -req -extfile <(printf "subjectAltName=DNS:$SERVICE.$NAMESPACE.svc,DNS:$SERVICE.$NAMESPACE.svc.cluster.local") \
    -days 365 -in $TMPDIR/server.csr -CA $TMPDIR/ca.crt -CAkey $TMPDIR/ca.key -CAcreateserial \
    -out $TMPDIR/tls.crt

# Create Kubernetes secret with the certificates
kubectl create secret generic $SECRET \
    --from-file=tls.key=$TMPDIR/tls.key \
    --from-file=tls.crt=$TMPDIR/tls.crt \
    --namespace=$NAMESPACE \
    --dry-run=client -o yaml | kubectl apply -f -

# Get the CA bundle for the webhook configuration
CA_BUNDLE=$(cat $TMPDIR/ca.crt | base64 | tr -d '\n')

echo "Certificates generated and secret created."
echo "CA Bundle (for webhook configuration): $CA_BUNDLE"

# Clean up
rm -rf $TMPDIR
echo "Temporary files cleaned up."