# Use the old Apple linker to workaround broken xcode - https://github.com/golang/go/issues/65169
ifeq ($(shell uname),Darwin)
	FUZZLDFLAGS := -ldflags=-extldflags=-Wl,-ld_classic
endif

op-version-check:
	go build -o ./bin/op-version-check ./cmd/op-version-check/main.go

ecotone-scalar:
	go build -o ./bin/ecotone-scalar ./cmd/ecotone-scalar/main.go

receipt-reference-builder:
	go build -o ./bin/receipt-reference-builder ./cmd/receipt-reference-builder/*.go

op-upgrade:
	go build -o ./bin/op-upgrade ./cmd/op-upgrade/main.go

test:
	go test ./...

fuzz:
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeWithdrawal ./crossdomain
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeLegacyWithdrawal ./crossdomain
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzAliasing ./crossdomain
	go test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzVersionedNonce ./crossdomain

.PHONY: test fuzz
