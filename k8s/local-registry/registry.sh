#!/usr/bin/env sh

set -e

IP="$1"
if [ -z "$IP" ]; then
  echo "Missing IP address that will be used for registry"
  exit 1
fi

docker run -d -p 5000:5000 --restart=always --name registry \
  -v $HOME/etc/vault.unsealer:/certs \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/services.pem \
  -e REGISTRY_HTTP_TLS_KEY=/certs/services-key.pem \
  registry:latest

docker tag hashicorp/consul-k8s:0.12.0 $IP:5000/hashicorp/consul-k8s:0.12.0
docker tag consul:1.7.1 $IP:5000/consul:1.7.1
docker tag omisego/immutability-vault-ethereum $IP:5000/omisego/immutability-vault-ethereum:1.0.0
docker tag vault:1.3.2 $IP:5000/vault:1.3.2

docker push $IP:5000/hashicorp/consul-k8s:0.12.0
docker push $IP:5000/consul:1.7.1
docker push $IP:5000/omisego/immutability-vault-ethereum:1.0.0
docker push $IP:5000/vault:1.3.2
docker image ls