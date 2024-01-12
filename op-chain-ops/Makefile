test:
	go test ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeWithdrawal ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzEncodeDecodeLegacyWithdrawal ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzAliasing ./crossdomain
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzVersionedNonce ./crossdomain

.PHONY: test fuzz
