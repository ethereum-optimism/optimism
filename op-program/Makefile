GITCOMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GITDATE ?= $(shell git show -s --format='%ct' 2>/dev/null || echo "0")

VERSION ?= v0.0.0

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)

# op-program-client version must ALWAYS be set to the same value (v0.0.0) to ensure exact build is reproducible
PC_LDFLAGSSTRING := $(LDFLAGSSTRING)
PC_LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Version=v0.0.0
PC_LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Meta=

LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Version=$(VERSION)
LDFLAGSSTRING +=-X github.com/ethereum-optimism/optimism/op-program/version.Meta=$(VERSION_META)

COMPAT_DIR := temp/compat

op-program: \
	op-program-host \
	op-program-client \
	op-program-client-mips

op-program-host:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -v -ldflags "$(LDFLAGSSTRING)" -o ./bin/op-program ./host/cmd/main.go

op-program-client:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v -ldflags "$(PC_LDFLAGSSTRING)" -o ./bin/op-program-client ./client/cmd/main.go
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v -ldflags "$(PC_LDFLAGSSTRING)" -o ./bin/op-program-client-interop ./client/interopcmd/main.go

op-program-client-mips: op-program-client-mips64 op-program-client-mips64-interop

op-program-client-mips64:
	env GO111MODULE=on GOOS=linux GOARCH=mips64 GOMIPS64=softfloat go build -v -ldflags "$(PC_LDFLAGSSTRING)" -o ./bin/op-program-client64.elf ./client/cmd/main.go
	# verify output with: readelf -h bin/op-program-client64.elf
	# result is mips64, big endian, R3000

op-program-client-mips64-interop:
	env GO111MODULE=on GOOS=linux GOARCH=mips64 GOMIPS64=softfloat go build -v -ldflags "$(PC_LDFLAGSSTRING)" -o ./bin/op-program-client-interop.elf ./client/interopcmd/main.go
	# verify output with: readelf -h bin/op-program-client-interop.elf
	# result is mips64, big endian, R3000

op-program-client-riscv:
	env GO111MODULE=on GOOS=linux GOARCH=riscv64 go build -v -gcflags="all=-d=softfloat" -ldflags "$(PC_LDFLAGSSTRING)" -o ./bin/op-program-client-riscv.elf ./client/cmd/main.go

check-custom-chains: op-program-host
	./bin/op-program configs check-custom-chains

reproducible-prestate: check-custom-chains
	@docker build --build-arg GO_VERSION=1.24.2-alpine3.21 --build-arg EXPORT_TARGET=next  --output ./bin/ --progress plain -f Dockerfile.repro ../
	@docker build --build-arg GO_VERSION=1.23.8-alpine3.21 --build-arg EXPORT_TARGET=current --output ./bin/ --progress plain -f Dockerfile.repro ../
	@echo "-------------------- Production Prestates --------------------"
	@echo "\nCannon64 Absolute prestate hash: "
	@cat ./bin/prestate-proof-mt64.json | jq -r .pre
	@echo "\n-------------------- Experimental Prestates --------------------"
	@echo "\nCannon64Next Absolute prestate hash: "
	@cat ./bin/prestate-proof-mt64Next.json | jq -r .pre
	@echo "\nCannonInterop Absolute prestate hash: "
	@cat ./bin/prestate-proof-interop.json | jq -r .pre
	@echo "\nCannonInteropNext Absolute prestate hash: "
	@cat ./bin/prestate-proof-interopNext.json | jq -r .pre
	@echo
.PHONY: reproducible-prestate

verify-reproducibility:
	rm -rf temp/states
	./scripts/build-prestates.sh
	env GO111MODULE=on go run ./prestates/verify/verify.go --input temp/states/versions.json
.PHONY: verify-reproducibility

clean:
	rm -rf bin "$(COMPAT_DIR)"

test:
	go test -v ./...

capture-mainnet-genesis: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/mainnet-genesis" "$(COMPAT_DIR)/mainnet-genesis.tar.bz"
	env GO111MODULE=on go run ./verify/mainnet/cmd/mainnet.go --l1 $$MAINNET_L1URL --l1.beacon $$MAINNET_BEACON_URL --l2 $$MAINNET_L2URL --datadir "$(COMPAT_DIR)/mainnet-genesis" --l1.head "0x4903424f6cc2cfba7c2bf8c8f48ca46721c963fa64b411cfee3697b781e3e5f1" --l2.start "105235063" --l2.end "105235064"
	tar jcf "$(COMPAT_DIR)/mainnet-genesis.tar.bz" -C "$(COMPAT_DIR)" mainnet-genesis

capture-sepolia-delta: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/sepolia-delta" "$(COMPAT_DIR)/sepolia-delta.tar.bz"
	env GO111MODULE=on go run ./verify/sepolia/cmd/sepolia.go --l1 $$SEPOLIA_L1URL --l1.beacon $$SEPOLIA_BEACON_URL --l2 $$SEPOLIA_L2URL --datadir "$(COMPAT_DIR)/sepolia-delta" --l1.head "0x935428728bcfcfeb2e5ba9175fd2890e52831dae221aa4d5dcffed8320edc001" --l2.start "8728200" --l2.end "8728320"
	tar jcf "$(COMPAT_DIR)/sepolia-delta.tar.bz" -C "$(COMPAT_DIR)" sepolia-delta

capture-sepolia-ecotone: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/sepolia-ecotone" "$(COMPAT_DIR)/sepolia-ecotone.tar.bz"
	env GO111MODULE=on go run ./verify/sepolia/cmd/sepolia.go --l1 $$SEPOLIA_L1URL --l1.beacon $$SEPOLIA_BEACON_URL --l2 $$SEPOLIA_L2URL --datadir "$(COMPAT_DIR)/sepolia-ecotone" --l1.head "0x5d491a8c1e728a4e70720c09fefdaa083681a9421cd365af85220cf8bd4448a3" --l2.start "9205715" --l2.end "9205815"
	tar jcf "$(COMPAT_DIR)/sepolia-ecotone.tar.bz" -C "$(COMPAT_DIR)" sepolia-ecotone

capture-sepolia-fjord: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/sepolia-fjord" "$(COMPAT_DIR)/sepolia-fjord.tar.bz"
	env GO111MODULE=on go run ./verify/sepolia/cmd/sepolia.go --l1 $$SEPOLIA_L1URL --l1.beacon $$SEPOLIA_BEACON_URL --l2 $$SEPOLIA_L2URL --datadir "$(COMPAT_DIR)/sepolia-fjord" --l1.head "0x93ba31bf89e54237af6e6564e69d328b2b5202adf643de4cb097431f74f4a6c1" --l2.start "15378256" --l2.end "15378356"
	tar jcf "$(COMPAT_DIR)/sepolia-fjord.tar.bz" -C "$(COMPAT_DIR)" sepolia-fjord

capture-sepolia-granite: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/sepolia-granite" "$(COMPAT_DIR)/sepolia-granite.tar.bz"
	env GO111MODULE=on go run ./verify/sepolia/cmd/sepolia.go --l1 $$SEPOLIA_L1URL --l1.beacon $$SEPOLIA_BEACON_URL --l2 $$SEPOLIA_L2URL --datadir "$(COMPAT_DIR)/sepolia-granite" --l1.head "0x4a4e2b07b1cb468f20b37de54308ec70d2a96453e5186b0bf0929bd5b63ca492" --l2.start "15837930" --l2.end "15838030"
	tar jcf "$(COMPAT_DIR)/sepolia-granite.tar.bz" -C "$(COMPAT_DIR)" sepolia-granite

capture-sepolia-holocene: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/sepolia-holocene" "$(COMPAT_DIR)/sepolia-holocene.tar.bz"
	env GO111MODULE=on go run ./verify/sepolia/cmd/sepolia.go --l1 $$SEPOLIA_L1URL --l1.beacon $$SEPOLIA_BEACON_URL --l2 $$SEPOLIA_L2URL --datadir "$(COMPAT_DIR)/sepolia-holocene" --l1.head "0x0fd505af2e97a0cf59232a8615340689ee4dc14d022103f6342ba4fd6b89f066" --l2.start "20415330" --l2.end "20415430"
	tar jcf "$(COMPAT_DIR)/sepolia-holocene.tar.bz" -C "$(COMPAT_DIR)" sepolia-holocene

capture-sepolia-isthmus: op-program-host op-program-client
	rm -rf "$(COMPAT_DIR)/sepolia-isthmus" "$(COMPAT_DIR)/sepolia-isthmus.tar.bz"
	env GO111MODULE=on go run ./verify/sepolia/cmd/sepolia.go --l1 $$SEPOLIA_L1URL --l1.beacon $$SEPOLIA_BEACON_URL --l2 $$SEPOLIA_L2URL --datadir "$(COMPAT_DIR)/sepolia-isthmus" --l1.head "0x6ae443d11c3896a458f80905601920f16e6ae6cb34f070db7307195f6e23e58c" --l2.start "26551530" --l2.end "26551630"
	tar jcf "$(COMPAT_DIR)/sepolia-isthmus.tar.bz" -C "$(COMPAT_DIR)" sepolia-isthmus

capture-chain-test-data: capture-mainnet-genesis capture-sepolia-delta capture-sepolia-ecotone capture-sepolia-fjord capture-sepolia-granite capture-sepolia-holocene capture-sepolia-isthmus

verify-mainnet-genesis: op-program-host op-program-client
	./scripts/run-compat.sh "mainnet-genesis"

verify-sepolia-delta: op-program-host op-program-client
	./scripts/run-compat.sh "sepolia-delta"

verify-sepolia-ecotone: op-program-host op-program-client
	./scripts/run-compat.sh "sepolia-ecotone"

verify-sepolia-fjord: op-program-host op-program-client
	./scripts/run-compat.sh "sepolia-fjord"

verify-sepolia-granite: op-program-host op-program-client
	./scripts/run-compat.sh "sepolia-granite"

verify-sepolia-holocene: op-program-host op-program-client
	./scripts/run-compat.sh "sepolia-holocene"

verify-sepolia-isthmus: op-program-host op-program-client
	./scripts/run-compat.sh "sepolia-isthmus"

verify-compat: verify-mainnet-genesis verify-sepolia-delta verify-sepolia-ecotone verify-sepolia-fjord verify-sepolia-granite verify-sepolia-holocene verify-sepolia-isthmus

analyze-op-program-client-current:
	./scripts/run-static-analysis.sh ./vm-profiles/cannon-multithreaded-64.yaml ./compatibility-test/baseline-cannon-multithreaded-64.json

analyze-op-program-client-next:
	./scripts/run-static-analysis.sh ./vm-profiles/cannon-multithreaded-64-next.yaml ./compatibility-test/baseline-cannon-multithreaded-64-next.json

run-vm-compat:
	@docker build --build-arg GO_VERSION=1.23.8-alpine3.21 --build-arg VM_TARGET=current --progress plain -f Dockerfile.vmcompat ../
	@docker build --build-arg GO_VERSION=1.24.2-alpine3.21 --build-arg VM_TARGET=next --progress plain -f Dockerfile.vmcompat ../

.PHONY: \
	op-program \
	op-program-host \
	op-program-client \
	op-program-client-mips \
	op-program-client-mips64 \
	op-program-client-riscv \
	clean \
	test \
	capture-mainnet-genesis \
	capture-sepolia-delta \
	capture-sepolia-ecotone \
	capture-sepolia-fjord \
	capture-sepolia-granite \
	capture-sepolia-holocene \
	capture-sepolia-isthmus \
	capture-chain-test-data \
	verify-mainnet-genesis \
	verify-sepolia-delta \
	verify-sepolia-ecotone \
	verify-sepolia-fjord \
	verify-sepolia-granite \
	verify-sepolia-holocene \
	verify-sepolia-isthmus \
	verify-compat \
	analyze-op-program-client \
	analyze-op-program-client-cannon-multithreaded-64
