all: check-l2

check-l2:
	go build -o ./bin/check-l2 ./cmd/check-l2/main.go

test:
	go test ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeWithdrawal ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeLegacyWithdrawal ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzAliasing ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzVersionedNonce ./crossdomain

.PHONY: check-l2 test fuzz
