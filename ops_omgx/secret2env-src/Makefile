.DEFAULT_GOAL := build

.PHONY: build
build:
	go get github.com/aws/aws-sdk-go && go install github.com/aws/aws-sdk-go && go build -o secret2env

.PHONY: build/linux
build/linux:
	env CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o secret2env
