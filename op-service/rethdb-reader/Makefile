.PHONY: testdata
testdata:
	mkdir -p testdata
	@echo "Fetching block RLP and receipts for block #18,663,292 from ethereum mainnet"
	cast rpc debug_getRawBlock 0x11CC77C | jq -r | xxd -r -p > testdata/block.rlp
	cast rpc debug_getRawReceipts 0x11CC77C | jq -r > testdata/receipts.json
	@echo "Done. Generating testdata DB & testing integrity..."
	cargo test
