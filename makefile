.ONESHELL:

IMG_VERSION = $(shell cat ./VERSION)
DATE = $(shell date +'%s')

docker-build:
	docker build --progress=plain --build-arg always_upgrade="$(DATE)" -t omgnetwork/vault:latest .
	docker tag omgnetwork/vault:latest omgnetwork/vault:$(IMG_VERSION)

test:
	docker-compose -f docker/docker-compose.yml up

run:
	docker-compose -f docker/lean-docker-compose.yml up

all: docker-build run

clean:
	mv docker/config/entrypoint.sh /tmp/entrypoint.sh
	mv docker/config/vault.hcl /tmp/vault.hcl
	rm -rf docker/config/*
	mv /tmp/entrypoint.sh docker/config/entrypoint.sh
	mv /tmp/vault.hcl docker/config/vault.hcl

abigen:
	docker run -d --rm -it --name ovm_contracts --entrypoint /bin/bash omgx/deployer:latest
	sleep 5s
	docker cp ovm_contracts:/opt/optimism/packages/contracts/artifacts /tmp/oc
	docker stop ovm_contracts
	docker run --rm -it --name abigen -v /tmp/oc:/tmp/oc --entrypoint /bin/sh ethereum/client-go:alltools-stable -c " apk add jq && cat /tmp/oc/contracts/optimistic-ethereum/OVM/chain/OVM_StateCommitmentChain.sol/OVM_StateCommitmentChain.json | jq .abi | abigen --abi - --pkg OVM_SCC --out /tmp/oc/ovm_scc.go && cat /tmp/oc/contracts/optimistic-ethereum/OVM/chain/OVM_CanonicalTransactionChain.sol/OVM_CanonicalTransactionChain.json | jq .abi | abigen --abi - --pkg OVM_CTC --out /tmp/oc/ovm_ctc.go && cat /tmp/oc/contracts/optimistic-ethereum/OVM/bridge/messaging/OVM_L1CrossDomainMessenger.sol/OVM_L1CrossDomainMessenger.json | jq .abi | abigen --abi - --pkg OVM_L1CDM --out /tmp/oc/ovm_l1cdm.go"
	cp /tmp/oc/*.go contracts/ovm

	