#!/bin/sh

set -e

# This script moves the Consul bootstrap token out of the Kubernetes secret 
# into the unsealer vault. 
#
# The inputs are:
#   The name of the Kubernetes secret containing the Bootstrap ACL Token
#   (Defaults to consul-backend-consul-bootstrap-acl-token)
# 
# Usage:
#   k8sclean.sh consul-backend-consul-bootstrap-acl-token

K8S_SECRET="$1"
if [ -z "$K8S_SECRET" ]; then
	K8S_SECRET="consul-backend-consul-bootstrap-acl-token"
fi

ACL_TOKEN=$(kubectl get secret $K8S_SECRET -o yaml | yq r - 'data.token' | base64 --decode)
if [ -z "$ACL_TOKEN" ]; then
	echo "Missing ACL_TOKEN"
	exit 1
fi

vault write kv/$K8S_SECRET value=$ACL_TOKEN
kubectl delete secret $K8S_SECRET