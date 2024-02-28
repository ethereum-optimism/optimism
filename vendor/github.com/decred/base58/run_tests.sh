#!/usr/bin/env bash

set -ex

# The script does automatic checking on a Go package and its sub-packages, including:
# 1. race detector (http://blog.golang.org/race-detector)
# 2. gofmt         (http://golang.org/cmd/gofmt/)
# 3. golint        (https://github.com/golang/lint)
# 4. go vet        (http://golang.org/cmd/vet)
# 5. gosimple      (https://github.com/dominikh/go-simple)
# 6. unconvert     (https://github.com/mdempsky/unconvert)
# 7. ineffassign   (https://github.com/gordonklaus/ineffassign)
# 8. misspell      (https://github.com/client9/misspell)
# 9. deadcode      (https://github.com/remyoudompheng/go-misc/tree/master/deadcode)

# run tests
env GORACE="halt_on_error=1" go test -race ./...

# golangci-lint (github.com/golangci/golangci-lint) is used to run each each
# static checker.

# check linters
golangci-lint run --disable-all --deadline=10m \
  --enable=gofmt \
  --enable=revive \
  --enable=vet \
  --enable=gosimple \
  --enable=unconvert \
  --enable=ineffassign \
  --enable=misspell \
  --enable=deadcode
