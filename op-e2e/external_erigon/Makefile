default: shim op-erigon

op-erigon:
	make -C ../../op-erigon
	cp ../../op-erigon/erigon op-erigon
.PHONY: op-erigon

shim: main.go
	go build -o shim .
