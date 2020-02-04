#!/bin/bash

set -x

kubectl apply -f k8s-agent-spec.yml --record
