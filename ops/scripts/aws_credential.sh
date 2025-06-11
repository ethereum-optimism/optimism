#!/bin/bash

AUDIENCE="KMSAccess"
ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/gke-kms-access-role"

jwt_token=$(curl -sH "Metadata-Flavor: Google" "http://metadata/computeMetadata/v1/instance/service-accounts/default/identity?audience=${AUDIENCE}&format=full&licenses=FALSE")
jwt_decoded=$(jq -R 'split(".") | .[1] | @base64d | fromjson' <<< "$jwt_token")
jwt_sub=$(echo -n "$jwt_decoded" | jq -r '.sub')

credentials=$(aws sts assume-role-with-web-identity \
  --role-arn "$ROLE_ARN" \
  --role-session-name "$jwt_sub" \
  --web-identity-token "$jwt_token" \
  | jq '.Credentials' \
  | jq '.Version=1')

echo "$credentials"
