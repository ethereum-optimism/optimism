build: unicorn minigeth_mips minigeth_default_arch mipsevm contracts
	yarn

unicorn:
	./build_unicorn.sh

minigeth_mips:
	cd mipigo && ./build.sh

minigeth_default_arch:
	cd minigeth && go build

mipsevm:
	cd mipsevm && go build

contracts:
	yarn
	npx hardhat compile

define clear_cache
	rm -rf /tmp/cannon
	mkdir -p /tmp/cannon
endef

test_challenge:
	$(call clear_cache)
	# Build preimage cache for block 13284469
	minigeth/go-ethereum 13284469
	# Generate initial (generic) MIPS memory checkpoint and final checkpoint for
	# block 13284469.
	mipsevm/mipsevm && mipsevm/mipsevm 13284469
	npx hardhat test test/challenge_test.js

test_mipsevm:
	$(call clear_cache)
	# Build preimage caches for the given blocks
	minigeth/go-ethereum 13284469
	minigeth/go-ethereum 13284491
	cd mipsevm && go test -v

test_minigeth:
	$(call clear_cache)
	# Check that minigeth is able to validate the given transactions.
	# run block 13284491 (0 tx)
	minigeth/go-ethereum 13284491
	# run block 13284469 (few tx)
	minigeth/go-ethereum 13284469
	# block 13284053 (deletion)
	minigeth/go-ethereum 13284053
	# run block 13303075 (uncles)
	minigeth/go-ethereum 13303075

test_contracts:
	npx hardhat test

test: test_challenge test_mipsevm test_minigeth

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
	clean mrproper test_challenge test_mipsevm test_minigeth test
