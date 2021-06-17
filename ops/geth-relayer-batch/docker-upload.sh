#!/bin/bash
set -ex
region='us-east-2'
if [ -n "$1"  ]
then
    region=$1
fi
# --no-cache
echo 'Building metis_l2_geth image'
cmd="sed -i s#REGION_VAR_FOR_ENV#$region#g ./settings/efs-utils.conf"
$cmd
# docker images|grep metis_l2_geth|awk '{print $3}'|xargs docker rmi -f
docker build --no-cache -f ./Dockerfile -t metis_l2_geth ../geth-relayer-batch
cmd="sed -i s#$region#REGION_VAR_FOR_ENV#g ./settings/efs-utils.conf"
$cmd

profile="aws --profile default ecr get-login-password --region $region"
login="docker login --username AWS --password-stdin 950087689901.dkr.ecr.$region.amazonaws.com"
$profile | $login

echo 'Pushing metis_l2_geth'
l2geth="docker tag metis_l2_geth:latest 950087689901.dkr.ecr.$region.amazonaws.com/metis-l2-geth:latest"
$l2geth
l2geth_push="docker push  950087689901.dkr.ecr.$region.amazonaws.com/metis-l2-geth:latest"
$l2geth_push

echo 'Pushing data-transport-layer'
dtl="docker tag ethereumoptimism/data-transport-layer:latest 950087689901.dkr.ecr.$region.amazonaws.com/metis-dtl:latest"
$dtl
dtl_push="docker push 950087689901.dkr.ecr.$region.amazonaws.com/metis-dtl:latest"
$dtl_push

echo 'Pushing data-transport-layer'
bs="docker tag ethereumoptimism/batch-submitter:latest 950087689901.dkr.ecr.$region.amazonaws.com/metis-l2-bs:latest"
$bs
bs_push="docker push 950087689901.dkr.ecr.$region.amazonaws.com/metis-l2-bs:latest"
$bs_push
