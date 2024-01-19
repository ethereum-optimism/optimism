# A simple make target to build erigon
.PHONY: erigon
erigon:
	@go build -tags nosqlite,noboltdb,nosilkworm github.com/ledgerwatch/erigon/cmd/erigon

.PHONY: clean
clean:
	@rm -f erigon
