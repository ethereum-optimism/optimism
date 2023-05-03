SHELL := /bin/bash

build: submodules libunicorn
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

clean:
	rm -f unicorn/libunicorn.*
.PHONY: clean

test:
	cd unicorntest && go test -v ./...
.PHONY: test
