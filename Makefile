opnode:
	go build -o ./bin/op ./opnode/cmd
.PHONY: opnode

contracts:
	cd ./packages/contracts && yarn build
.PHONY: contracts

clean:
	rm -rf ./bin
.PHONY: clean

devnet-clean: devnet-down
	cd ./ops && docker-compose rm
	docker volume rm ops_l1_data
	docker volume rm ops_l2_data
.PHONY: devnet-clean

devnet-up:
	@(cd ./ops && \
		DEPOSIT_FEED_BYTECODE=$(shell cat ./packages/contracts/artifacts/contracts/L1/DepositFeed.sol/DepositFeed.json | jq .deployedBytecode) \
			L1_BLOCK_INFO_BYTECODE=$(shell cat ./packages/contracts/artifacts/contracts/L2/L1Block.sol/L1Block.json | jq .deployedBytecode) \
			WITHDRAWOR_BYTECODE=$(shell cat ./packages/contracts/artifacts/contracts/L2/Withdrawor.sol/Withdrawor.json | jq .deployedBytecode) \
 			docker-compose up --build)
.PHONY: devnet-up

devnet-down:
	@(cd ./ops && docker-compose down)
.PHONY: devnet-stop
