PROTOS := ./proto/src
PROTO_GEN := ./proto/gen

test:
	go test -v ./...

generate-mocks:
	go generate ./...

.PHONY: \
	test \
	lint \
	generate-mocks \
	fuzz \
	clean \
	protoc

fuzz:
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadUnmarshal ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshalV1 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshalV2 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzExecutionPayloadMarshalUnmarshalV3 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzOBP01 ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzEncodeDecodeBlob ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzDetectNonBijectivity ./eth
	go test -run NOTAREALTEST -v -fuzztime 10s -fuzz FuzzEncodeScalar ./eth

clean:
	find $(PROTO_GEN) -name "*.pb.go" -type f | xargs rm -rf
	mkdir -p $(PROTO_GEN)

protoc: clean
	protoc -I $(PROTOS) \
	--go_out=$(PROTO_GEN) \
	--go_opt=paths=source_relative \
	--go-grpc_out=$(PROTO_GEN) \
	--go-grpc_opt=paths=source_relative \
	$$(find $(PROTOS) -name '*.proto')
