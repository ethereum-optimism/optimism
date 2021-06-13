#!/bin/bash
set -ex

# --no-cache
echo 'Building metis_l2_geth image'
cmd="sed -i s#REGION_VAR_FOR_ENV#$1#g  ./settings/efs-utils.conf"
$cmd
#docker images|grep metis_l2_geth|awk '{print $3}'|xargs docker rmi -f
docker build --no-cache -f ./Dockerfile -t metis_l2_geth ../geth-relayer-batch

profile="aws --profile default ecr get-login-password --region $1"
login="docker login --username AWS --password-stdin 950087689901.dkr.ecr.$1.amazonaws.com"
$profile | $login

echo 'Pushing metis_l2_geth'
l2geth="docker tag metis_l2_geth:latest 950087689901.dkr.ecr.$1.amazonaws.com/metis-l2-geth:latest"
$l2geth
l2geth_push="docker push  950087689901.dkr.ecr.$1.amazonaws.com/metis-l2-geth:latest"
$l2geth_push

echo 'Pushing data-transport-layer'
dtl="docker tag ethereumoptimism/data-transport-layer:latest 950087689901.dkr.ecr.$1.amazonaws.com/metis-dtl:latest"
$dtl
dtl_pupsh="docker push 950087689901.dkr.ecr.$1.amazonaws.com/metis-dtl:latest"
$dtl_pupsh