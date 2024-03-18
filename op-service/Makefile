test:
	go test -v ./...

generate-mocks:
	go generate ./...

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadUnmarshal ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshalV1 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshalV2 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshalV3 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzOBP01 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzEncodeDecodeBlob ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDetectNonBijectivity ./eth

.PHONY: \
	test \
	generate-mocks \
	fuzz
