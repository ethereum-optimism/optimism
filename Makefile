build: unicorn minigeth_mips minigeth_default_arch mipsevm contracts
	yarn

unicorn:
	./build_unicorn.sh

minigeth_mips:
	(cd mipigo && ./build.sh)

minigeth_default_arch:
	(cd minigeth && go build)

mipsevm:
	(cd mipsevm && go build)

contracts:
	yarn
	npx hardhat compile

clean:
	rm minigeth/go-ethereum
	rm mipigo/minigeth
	rm mipigo/minigeth.bin
	rm mipsevm/mipsevm
	rm -rf artifacts

mrproper: clean
	rm -rf cache
	rm -rf node_modules
	rm -rf mipigo/venv

.PHONY: build unicorn minigeth_mips minigeth_default_arch mipsevm contracts \
	clean mrproper
