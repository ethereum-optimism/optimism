GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

# Use the old Apple linker to workaround broken xcode - https://github.com/golang/go/issues/65169
ifeq ($(shell uname),Darwin)
	FUZZLDFLAGS := -ldflags=-extldflags=-Wl,-ld_classic
endif

cannon:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/cannon .

clean:
	rm -rf bin

elf:
	make -C ./testdata/example elf

contract:
	cd ../packages/contracts-bedrock && forge build

test: elf contract
	go test -v ./...

fuzz:
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallBrk ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallClone ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallMmap ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallExitGroup ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallFcntl ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateHintRead ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 20s -fuzz=FuzzStatePreimageRead ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateHintWrite ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 20s -fuzz=FuzzStatePreimageWrite ./mipsevm/tests

.PHONY: \
	cannon \
	clean \
	test \
	lint \
	fuzz
