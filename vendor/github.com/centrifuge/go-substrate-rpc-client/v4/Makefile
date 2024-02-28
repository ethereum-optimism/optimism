# Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
#
# Copyright 2019 Centrifuge GmbH
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
clean:				## cleanup
	@rm -f coverage.txt
	@docker-compose down

lint:				## run linters on go code
	@docker run -v `pwd`:/app -w /app golangci/golangci-lint:v1.45.2 golangci-lint run

lint-fix: 			## run linters on go code and automatically fixes issues
	@docker run -v `pwd`:/app -w /app golangci/golangci-lint:v1.45.2 golangci-lint run --fix

test: 				## run all tests in project against the RPC URL specified in the RPC_URL env variable or localhost while excluding gethrpc
	@go test -race -count=1 `go list ./... | grep -v '/gethrpc'`

test-cover: 			## run all tests in project against the RPC URL specified in the RPC_URL env variable or localhost and report coverage
	@go test -race -coverprofile=coverage.txt -covermode=atomic `go list ./... | grep -v '/gethrpc'`

test-dockerized:		## run all tests in a docker container against the Substrate Default Docker image
test-dockerized: run-substrate-docker
	@sleep 15
	@docker-compose build; docker-compose up --abort-on-container-exit gsrpc-test

run-substrate-docker: 		## starts the Substrate Docker image
	@docker-compose up -d substrate

generate-test-data:		## generate data for types decode test
	@go generate -tags=types_test ./types/test/...

test-types-decode:      ## run tests for types decode
	@go test -tags=types_test ./types/test/...

generate-mocks:      ## generate mocks
	@docker run -v `pwd`:/app -w /app --entrypoint /bin/sh vektra/mockery:v2.13.0-beta.1 -c 'go generate ./...'

help: 				## shows this help
	@sed -ne '/@sed/!s/## //p' $(MAKEFILE_LIST)

.PHONY: install-deps lint lint-fix test test-cover test-dockerized run-substrate-docker clean generate-test-data
