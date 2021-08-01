yarn 
yarn build

docker-compose build -- builder
docker-compose build -- omgx_deployer
docker-compose build -- omgx_message-relayer-fast
docker-compose build -- gas_oracle
docker-compose build -- deployer

docker rmi $(docker images -f "dangling=true" -q)

docker-compose build -- l2geth 
docker-compose build -- l1_chain
docker-compose build -- batch_submitter
docker-compose build -- dtl
docker-compose build -- relayer
docker-compose build -- integration_tests

docker rmi $(docker images -f "dangling=true" -q)

docker ps

wait