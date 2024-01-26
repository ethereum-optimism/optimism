op-version-check:
	go build -o ./bin/op-version-check ./cmd/op-version-check/main.go

ecotone-scalar:
	go build -o ./bin/ecotone-scalar ./cmd/ecotone-scalar/main.go

receipt-reference-builder:
	go build -o ./bin/receipt-reference-builder ./cmd/receipt-reference-builder/main.go

test:
	go test ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeWithdrawal ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeLegacyWithdrawal ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzAliasing ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzVersionedNonce ./crossdomain

.PHONY: test fuzz
