GIT_COMMIT := ""
GIT_DATE := ""

CANNON_VERSION := "v0.0.0"
OP_PROGRAM_VERSION := "v0.0.0"

GOOS := ""
GOARCH := ""

# Build the cannon binary
cannon:
    #!/bin/bash
    # in devnet scenario, the cannon binary is already built.
    [ -x /app/cannon/bin/cannon ] && exit 0
    cd ../cannon
    env GOOS={{GOOS}} GOARCH={{GOARCH}} \
        just GITCOMMIT={{GIT_COMMIT}} GITDATE={{GIT_DATE}} VERSION={{CANNON_VERSION}} cannon

# Build the op-program-client elf binaries
# Note: GOOS/GOARCH/GOMIPS are hardcoded in the justfile targets,
# so they don't need to be passed here.
op-program-client-mips:
    #!/bin/bash
    cd ../op-program
    just GITCOMMIT={{GIT_COMMIT}} GITDATE={{GIT_DATE}} VERSION={{OP_PROGRAM_VERSION}} op-program-client-mips

# Generate the prestate proof containing the absolute pre-state hash.
prestate TYPE CLIENT_SUFFIX PRESTATE_SUFFIX: cannon op-program-client-mips
    #!/bin/bash
    go run /app/op-program/builder/main.go build-prestate \
        --program-elf /app/op-program/bin/op-program-client{{CLIENT_SUFFIX}}.elf \
        --version {{TYPE}}\
        --suffix {{PRESTATE_SUFFIX}}

build-mt64: (prestate "multithreaded64-5" "64" "-mt64")
build-mt64Next: (prestate "multithreaded64-5" "64" "-mt64Next")
build-interop: (prestate "multithreaded64-5" "-interop" "-interop")
build-interopNext: (prestate "multithreaded64-5" "-interop" "-interopNext")

build-current: build-mt64 build-interop
build-next: build-mt64Next build-interopNext
