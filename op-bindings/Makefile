SHELL := /bin/bash

all: mkdir bindings deployed

bindings: l1block-bindings l2-to-l1-message-passer-bindings optimism-portal-bindings l2-output-oracle-bindings gas-price-oracle-bindings address-manager-bindings
deployed: l1-block-deployed optimism-portal-deployed l2-to-l1-message-passer-deployed gas-price-oracle-deployed

l1-block-deployed: l1block-bindings
	./gen_deployed_bytecode.sh L1Block bindings

optimism-portal-deployed: optimism-portal-bindings
	./gen_deployed_bytecode.sh OptimismPortal bindings

l2-to-l1-message-passer-deployed: l2-to-l1-message-passer-bindings
	./gen_deployed_bytecode.sh L2ToL1MessagePasser bindings

gas-price-oracle-deployed: gas-price-oracle-bindings
	./gen_deployed_bytecode.sh GasPriceOracle bindings

l1block-bindings:
	./gen_bindings.sh L1Block bindings

l2-to-l1-message-passer-bindings:
	./gen_bindings.sh L2ToL1MessagePasser bindings

optimism-portal-bindings:
	./gen_bindings.sh OptimismPortal bindings

l2-output-oracle-bindings:
	./gen_bindings.sh L2OutputOracle bindings

gas-price-oracle-bindings:
	./gen_bindings.sh GasPriceOracle bindings

address-manager-bindings:
	./gen_bindings.sh AddressManager bindings

mkdir:
	mkdir -p bin bindings

clean:
	rm -rf bin bindings
