GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/cannon/multicannon/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/cannon/multicannon/version.Meta=$(VERSION_META)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

# Use the old Apple linker to workaround broken xcode - https://github.com/golang/go/issues/65169
ifeq ($(shell uname),Darwin)
	FUZZLDFLAGS := -ldflags=-extldflags=-Wl,-ld_classic
endif

cannon-impl:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/cannon-impl .

cannon-embeds: cannon-impl
	@cp bin/cannon-impl ./multicannon/embeds/cannon-0
	@cp bin/cannon-impl ./multicannon/embeds/cannon-1

cannon: cannon-embeds
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/cannon ./multicannon/

clean:
	rm -rf bin

elf:
	make -C ./testdata/example elf

sanitize-program:
	@if ! { mips-linux-gnu-objdump -d -j .text $$GUEST_PROGRAM | awk '{print $3}' | grep -Ew -m1 '(bgezal|bltzal)'; }; then \
		echo "guest program is sanitized for unsupported instructions"; \
	else \
		echo "found unsupported instructions in the guest program"; \
		exit 1; \
	fi

contract:
	cd ../packages/contracts-bedrock && forge build

test: elf contract
	go test -v ./...

fuzz:
  # Common vm tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallBrk ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallMmap ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallExitGroup ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallFcntl ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateHintRead ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 20s -fuzz=FuzzStatePreimageRead ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateHintWrite ./mipsevm/tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 20s -fuzz=FuzzStatePreimageWrite ./mipsevm/tests
	# Single-threaded tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallCloneST ./mipsevm/tests
	# Multi-threaded tests
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzStateSyscallCloneMT ./mipsevm/tests

.PHONY: \
	cannon \
	clean \
	test \
	lint \
	fuzz
