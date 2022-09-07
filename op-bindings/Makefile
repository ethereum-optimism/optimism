SHELL := /bin/bash

pkg := bindings

all: mkdir bindings deployed

bindings: l1block-bindings \
	l1-cross-domain-messenger-bindings \
  	l1-standard-bridge-bindings \
	l2-to-l1-message-passer-bindings \
	optimism-portal-bindings \
	l2-output-oracle-bindings \
	gas-price-oracle-bindings \
	address-manager-bindings \
	l2-cross-domain-messenger-bindings \
	l2-standard-bridge-bindings \
	sequencer-fee-vault-bindings \
	optimism-mintable-erc20-factory-bindings \
	optimism-mintable-erc20-bindings \
	proxy-bindings \
	proxy-admin-bindings \
	erc20-bindings \
	weth9-bindings

deployed: l1-block-deployed \
	optimism-portal-deployed \
	l2-to-l1-message-passer-deployed \
	gas-price-oracle-deployed

l1-block-deployed: l1block-bindings
	./gen_deployed_bytecode.sh L1Block $(pkg)

l1-cross-domain-messenger-bindings:
	./gen_bindings.sh contracts/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger $(pkg)

l1-standard-bridge-bindings:
	./gen_bindings.sh contracts/L1/L1StandardBridge.sol:L1StandardBridge $(pkg)

optimism-portal-deployed: optimism-portal-bindings
	./gen_deployed_bytecode.sh OptimismPortal $(pkg)

l2-to-l1-message-passer-deployed: l2-to-l1-message-passer-bindings
	./gen_deployed_bytecode.sh L2ToL1MessagePasser $(pkg)

gas-price-oracle-deployed: gas-price-oracle-bindings
	./gen_deployed_bytecode.sh GasPriceOracle $(pkg)

optimism-portal-bindings:
	./gen_bindings.sh contracts/L1/OptimismPortal.sol:OptimismPortal $(pkg)

l2-output-oracle-bindings:
	./gen_bindings.sh contracts/L1/L2OutputOracle.sol:L2OutputOracle $(pkg)

address-manager-bindings:
	./gen_bindings.sh contracts/legacy/AddressManager.sol:AddressManager $(pkg)

l1block-bindings:
	./gen_bindings.sh contracts/L2/L1Block.sol:L1Block $(pkg)

l2-to-l1-message-passer-bindings:
	./gen_bindings.sh contracts/L2/L2ToL1MessagePasser.sol:L2ToL1MessagePasser $(pkg)

gas-price-oracle-bindings:
	./gen_bindings.sh contracts/L2/GasPriceOracle.sol:GasPriceOracle $(pkg)

l2-cross-domain-messenger-bindings:
	./gen_bindings.sh contracts/L2/L2CrossDomainMessenger.sol:L2CrossDomainMessenger $(pkg)

l2-standard-bridge-bindings:
	./gen_bindings.sh contracts/L2/L2StandardBridge.sol:L2StandardBridge $(pkg)

sequencer-fee-vault-bindings:
	./gen_bindings.sh contracts/L2/SequencerFeeVault.sol:SequencerFeeVault $(pkg)

optimism-mintable-erc20-factory-bindings:
	./gen_bindings.sh contracts/universal/OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory $(pkg)

optimism-mintable-erc20-bindings:
	./gen_bindings.sh contracts/universal/OptimismMintableERC20.sol:OptimismMintableERC20 $(pkg)

proxy-bindings:
	./gen_bindings.sh contracts/universal/Proxy.sol:Proxy $(pkg)

proxy-admin-bindings:
	./gen_bindings.sh contracts/universal/ProxyAdmin.sol:ProxyAdmin $(pkg)

erc20-bindings:
	./gen_bindings.sh ERC20 $(pkg)

weth9-bindings:
	./gen_bindings.sh contracts/vendor/WETH9.sol:WETH9 $(pkg)

mkdir:
	mkdir -p bin $(pkg)

clean:
	rm -rf bin $(pkg)
