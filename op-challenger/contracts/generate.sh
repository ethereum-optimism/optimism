#! /bin/bash

cd project
forge inspect IL2OutputOracle abi > ../abi/l2OutputOracle.abi
forge inspect IL2OutputOracle bytecode > ../bin/l2OutputOracle.bin
forge inspect MockAttestationDisputeGame abi > ../abi/mockAttestationDisputeGame.abi
forge inspect MockAttestationDisputeGame bytecode > ../bin/mockAttestationDisputeGame.bin
forge inspect MockDisputeGameFactory abi > ../abi/mockDisputeGameFactory.abi
forge inspect MockDisputeGameFactory bytecode > ../bin/mockDisputeGameFactory.bin
cd ..

abigen \
	--abi abi/l2OutputOracle.abi \
	--bin bin/l2OutputOracle.bin \
	--pkg bindings \
    --type L2OutputOracle \
	--out ./bindings/l2OutputOracle.go

abigen \
	--abi abi/mockAttestationDisputeGame.abi \
	--bin bin/mockAttestationDisputeGame.bin \
	--pkg bindings \
    --type MockAttestationDisputeGame \
	--out ./bindings/mockAttestationDisputeGame.go

abigen \
	--abi abi/mockDisputeGameFactory.abi \
	--bin bin/mockDisputeGameFactory.bin \
	--pkg bindings \
    --type MockDisputeGameFactory \
	--out ./bindings/mockDisputeGameFactory.go