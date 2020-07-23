#!/usr/bin/env bash

: "$1?'missing key directory'"

key_dir="$1"

chmod 0700 "$key_dir"
cd "$key_dir"

[ -z "$service" ] && service=king-preset
[ -z "$namespace" ] && namespace=kingfisher

# 生成CA证书和CA私钥
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=Admission Controller Webhook Kingfisher"  -days 36500
# 生成Webhook服务的私钥
openssl genrsa -out webhook-server-tls.key 2048
# 为Webhook服务的私钥生成证书签名请求(CSR)，并使用CA的私钥对其进行签名
openssl req -new -key webhook-server-tls.key -subj "/CN=${service}.${namespace}.svc"  \
    | openssl x509 -req -CA ca.crt -CAkey ca.key -CAcreateserial -out webhook-server-tls.crt -days 36500