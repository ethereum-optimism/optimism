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

.DEFAULT_GOAL := cannon

# The MIPS64 r2 opcodes not supported by cannon. This list does not include coprocess-specific and trap opcodes.
UNSUPPORTED_OPCODES := (dclo|dclz|madd|maddu|seb|seh|wsbh|dsbh|dshd|ins|dins|dinsm|dinsu|ext|dext|dextu|dextm|rotr|drotr|drotr32|rotrv|drotrv|break|sdbbp|pref)

CANNON64_FUZZTIME := 20s

cannon64-impl:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/cannon64-impl .

# Note: This target is used by ./scripts/build-legacy-cannons.sh
# It should build the individual versions of cannons and copy them into place in the multicannon/embeds directory
# Ideally, preserve backwards compatibility with this behaviour but if it needs to change, build-legacy-cannons.sh will
# need to be updated to account for different behaviours in different versions.
# Each embed is suffixed with the latest `StateVersion` number corresponding to the target VM and architecture.
cannon-embeds: cannon64-impl
	# 64-bit multithreaded vm
	@cp bin/cannon64-impl ./multicannon/embeds/cannon-7
	@cp bin/cannon64-impl ./multicannon/embeds/cannon-8

cannon: cannon-embeds
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/cannon ./multicannon/

clean:
	rm -rf bin multicannon/embeds/cannon*

elf:
	make -C ./testdata elf

sanitize-program:
	mips-linux-gnu-objdump -d -j .text $$GUEST_PROGRAM > ./bin/dump.txt
	@if ! { cat ./bin/dump.txt | awk '{print $$3}' | grep -Ew -m1 "$(UNSUPPORTED_OPCODES)"; }; then \
		echo "guest program is sanitized for unsupported instructions"; \
	else \
		echo "found unsupported instructions in the guest program"; \
		exit 1; \
	fi

contract:
	cd ../packages/contracts-bedrock && forge build

test: elf contract
	go test -v ./...


diff-%-cannon: cannon elf
	# Load an elf file to create a prestate, and check that both cannon versions generate the same prestate
	@VM=$*; \
	echo "Running diff for VM type $${VM}"; \
	$$OTHER_CANNON load-elf --type $$VM --path ./testdata/go-1-23/bin/hello.64.elf --out ./bin/prestate-other.bin.gz --meta ""; \
	./bin/cannon   load-elf --type $$VM --path ./testdata/go-1-23/bin/hello.64.elf --out ./bin/prestate.bin.gz --meta "";
	@cmp ./bin/prestate-other.bin.gz ./bin/prestate.bin.gz;
	@if [ $$? -eq 0 ]; then \
		echo "Generated identical prestates"; \
	else \
		echo "Generated different prestates"; \
		exit 1; \
	fi

  # Run cannon and check that both cannon versions produce identical states
	$$OTHER_CANNON run --proof-at '=0' --stop-at '=100000000' --input=./bin/prestate.bin.gz  --output ./bin/out-other.bin.gz --meta ""
	./bin/cannon   run --proof-at '=0' --stop-at '=100000000' --input=./bin/prestate.bin.gz  --output ./bin/out.bin.gz --meta ""
	@cmp ./bin/out-other.bin.gz ./bin/out.bin.gz
	@if [ $$? -eq 0 ]; then \
		echo "Generated identical post-states"; \
	else \
		echo "Generated different post-states"; \
		exit 1; \
	fi

cannon-stf-verify:
	@docker build --progress plain -f Dockerfile.diff ../

fuzz:
	printf "%s\n" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateConsistencyMulOp ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateConsistencyMultOp ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateConsistencyMultuOp ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateSyscallBrk ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateSyscallMmap ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateSyscallExitGroup ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateSyscallFcntl ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateHintRead ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStatePreimageRead ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateHintWrite ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStatePreimageWrite ./mipsevm/tests" \
		"go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime $(CANNON64_FUZZTIME) -fuzz=FuzzStateSyscallCloneMT ./mipsevm/tests" \
	| parallel -j 8 {}

.PHONY: \
	cannon64-impl \
	cannon-embeds \
	cannon \
	clean \
	elf \
	elf-go-123 \
	test \
	lint \
	fuzz \
	diff-%-cannon \
	cannon-stf-verify
