#!/usr/bin/env sh

IP="$1"
if [ -z "$IP" ]; then
  echo "Missing IP address that will be used for registry"
  exit 1
fi

docker rmi $IP:5000/hashicorp/consul-k8s:0.12.0
docker rmi $IP:5000/consul:1.7.1
docker rmi $IP:5000/omisego/immutability-vault-ethereum:1.0.0
