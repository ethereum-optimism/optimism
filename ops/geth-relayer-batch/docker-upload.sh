#!/bin/bash
set -ex

# --no-cache
echo 'Building metis_l2_geth image'
#docker images|grep metis_l2_geth|awk '{print $3}'|xargs docker rmi -f
image=$(docker build --no-cache -f ./Dockerfile -t metis_l2_geth ../geth-relayer-batch)

ecr_login=$(aws --profile default ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin 950087689901.dkr.ecr.us-east-2.amazonaws.com)

echo 'Pushing metis_l2_geth'
tag=$(docker tag metis_l2_geth:latest 950087689901.dkr.ecr.us-east-2.amazonaws.com/metis-l2-geth:latest)
push=$(docker push  950087689901.dkr.ecr.us-east-2.amazonaws.com/metis-l2-geth:latest)

echo 'Pushing data-transport-layer'
tag=$(docker tag ethereumoptimism/data-transport-layer:latest 950087689901.dkr.ecr.us-east-2.amazonaws.com/metis-dtl:latest)
push=$(docker push 950087689901.dkr.ecr.us-east-2.amazonaws.com/metis-dtl:latest)