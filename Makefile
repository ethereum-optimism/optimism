SHELL := /bin/bash

build: submodules libunicorn minigeth_mips minigeth_prefetch mipsevm contracts
.PHONY: build

submodules:
	# CI will checkout submodules on its own (and fails on these commands)
	if [[ -z "$$GITHUB_ENV" ]]; then \
		git submodule init; \
		git submodule update; \
	fi
.PHONY: submodules

# Approximation, use `make libunicorn_rebuild` to force.
unicorn/build: unicorn/CMakeLists.txt
	mkdir -p unicorn/build
	cd unicorn/build && cmake .. -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release
	# Not sure why, but the second invocation is needed for fresh installs on MacOS.
	if [ "$(shell uname)" == "Darwin" ]; then \
		cd unicorn/build && cmake .. -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release; \
	fi

# Rebuild whenever anything in the unicorn/ directory changes.
unicorn/build/libunicorn.so: unicorn/build unicorn
	cd unicorn/build && make -j8
	# The Go linker / runtime expects dynamic libraries in the unicorn/ dir.
	find ./unicorn/build -name "libunicorn.*" | xargs -L 1 -I {} cp {} ./unicorn/
	# Update timestamp on libunicorn.so to make it more recent than the build/ dir.
	# On Mac this will create a new empty file (dyn libraries are .dylib), but works
	# fine for the purpose of avoiding recompilation.
	touch unicorn/build/libunicorn.so

libunicorn: unicorn/build/libunicorn.so
.PHONY: libunicorn

libunicorn_rebuild:
	touch unicorn/CMakeLists.txt
	make libunicorn
.PHONY: libunicorn_rebuild

minigeth_mips:
	cd mipigo && ./build.sh
.PHONY: minigeth_mips

minigeth_prefetch:
	cd minigeth && go build
.PHONY: minigeth_prefetch

mipsevm:
	cd mipsevm && go build
.PHONY: mipsevm

contracts: nodejs
	npx hardhat compile
.PHONY: contracts

nodejs:
	if [ -x "$$(command -v pnpm)" ]; then \
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
	rm -f unicorn/libunicorn.*
.PHONY: clean

mrproper: clean
	rm -rf cache
	rm -rf node_modules
	rm -rf mipigo/venv
	rm -rf unicorn/build
.PHONY:  mrproper
