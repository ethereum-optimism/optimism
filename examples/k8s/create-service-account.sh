#!/bin/bash

set -x


# Create a service account, 'omisego-service'
kubectl create serviceaccount omisego-service

# Update the 'omisego-service' service account
kubectl apply --filename omisego-service-account.yml
