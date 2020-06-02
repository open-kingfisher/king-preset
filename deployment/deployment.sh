#!/usr/bin/env bash


# Generate keys into a temporary directory.
echo "Generating TLS keys ..."
"./webhook-generate-keys.sh" "keys/"

# Create the TLS secret for the generated keys.
echo "Creating secret ..."
kubectl -n kingfisher create secret tls king-preset \
    --cert "keys/webhook-server-tls.crt" \
    --key "keys/webhook-server-tls.key"

echo "Deployment ..."
ca_pem_b64="$(openssl base64 -A < "keys/ca.crt")"
sed -e 's@${CA_PEM_B64}@'"$ca_pem_b64"'@g' <"deployment_all_in_one.yaml" \
    | kubectl create -f -