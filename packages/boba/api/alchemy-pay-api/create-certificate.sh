#!/bin/bash

# Request a new certificate for the domain
aws acm request-certificate \
  --domain-name "*.sepolia.boba.network" \
  --validation-method DNS \
  --region us-east-1

# After running this, you'll need to:
# 1. Get the CNAME records for validation:
# aws acm describe-certificate --certificate-arn <ARN_FROM_ABOVE_COMMAND> --region us-east-1

# 2. Add the CNAME records to your Route53 hosted zone or DNS provider
# 3. Wait for validation to complete (can take up to 30 minutes)
