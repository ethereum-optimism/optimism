SHELL := /bin/bash

build: submodules unicorn minigeth_mips minigeth_default_arch mipsevm contracts
.PHONY: build

# Approximation, use `make unicorn_rebuild` to force.
unicorn/build: unicorn/CMakeLists.txt
	mkdir -p unicorn/build
	cd unicorn/build && cmake .. -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release

unicorn: unicorn/build
	cd unicorn/build && make -j8
	# The Go linker / runtime expects these to be there!
	cp unicorn/build/libunicorn.so.1 unicorn
	cp unicorn/build/libunicorn.so.2 unicorn
.PHONY: unicorn

unicorn_rebuild:
	touch unicorn/CMakeLists.txt
	make unicorn
.PHONY: unicorn_rebuild

submodules:
	# CI will checkout submodules on its own (and fails on these commands)
	if [[ -z "$$GITHUB_ENV" ]]; then \
		git submodule init; \
		git submodule update; \
	fi
.PHONY: submodules

minigeth_mips:
	cd mipigo && ./build.sh
.PHONY: minigeth_mips

minigeth_default_arch:
	cd minigeth && go build
.PHONY: minigeth_default_arch

mipsevm:
	cd mipsevm && go build
.PHONY: mipsevm

contracts: nodejs
	npx hardhat compile
.PHONY: contracts

nodejs:
	if [ -x "$(command -v pnpm)" ]; then \
		pnpm install; \
	else \
		npm install; \
	fi
.PHONY: nodejs

# Must be a definition and not a rule, otherwise it gets only called once and
# not before each test as we wish.
define clear_cache
	rm -rf /tmp/cannon
	mkdir -p /tmp/cannon
endef

clear_cache:
	$(call clear_cache)
.PHONY: clear_cache

test_challenge:
	$(call clear_cache)
	# Build preimage cache for block 13284469
	minigeth/go-ethereum 13284469
	# Generate initial (generic) MIPS memory checkpoint and final checkpoint for
	# block 13284469.
	mipsevm/mipsevm && mipsevm/mipsevm 13284469
	npx hardhat test test/challenge_test.js
.PHONY: test_challenge

test_mipsevm:
	$(call clear_cache)
	# Build preimage caches for the given blocks
	minigeth/go-ethereum 13284469
	minigeth/go-ethereum 13284491
	cd mipsevm && go test -v
.PHONY: test_mipsevm

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
.PHONY: test_minigeth

test_contracts:
	$(call clear_cache)
	npx hardhat test
.PHONY: test_contracts

test: test_challenge test_mipsevm test_minigeth
.PHONY: test

clean:
	rm -f minigeth/go-ethereum
	rm -f mipigo/minigeth
	rm -f mipigo/minigeth.bin
	rm -f mipsevm/mipsevm
	rm -rf artifacts
.PHONY: clean

mrproper: clean
	rm -rf cache
	rm -rf node_modules
	rm -rf mipigo/venv
.PHONY:  mrproper
