#!/bin/sh


function banner {
    echo "------------------------------------------------------------------------------------------------------------------------------------"
}

source /vault/scripts/smoke.env.sh

echo 'should work with more complex case'
#response=$(curl -k -H X-Vault-Token:test-root-token -X PUT https://localhost:8200/v1/immutability-eth-plugin/encodeAppendSequencerBatch -d '{"should_start_at_element": 10, "total_elements_to_append": 1, "contexts": ["{\"num_sequenced_transactions\": 2, \"num_subsequent_queue_transactions\": 1, \"timestamp\": 100, \"block_number\": 200}"], "transactions": ["0x45423400000011", "0x45423400000012"]}' | jq .data)
response=$(vault write -f -field=data immutability-eth-plugin/encodeAppendSequencerBatch should_start_at_element=10 total_elements_to_append=1 contexts="{\"num_sequenced_transactions\": 2, \"num_subsequent_queue_transactions\": 1, \"timestamp\": 100, \"block_number\": 200}" transactions="0x45423400000011" transactions="0x45423400000012")
expected="000000000a000001000001000002000001000000006400000000c80000074542340000001100000745423400000012"
check_string_result $response $expected

echo 'should work with the simple case'
#response=$(curl -k -H X-Vault-Token:test-root-token -X PUT https://localhost:8200/v1/immutability-eth-plugin/encodeAppendSequencerBatch -d '{"should_start_at_element": 0, "total_elements_to_append": 0, "contexts": [], "transactions": []}' | jq .data)
response=$(vault write -f -field=data immutability-eth-plugin/encodeAppendSequencerBatch should_start_at_element=0 total_elements_to_append=0 contexts="" transactions="")
expected="0000000000000000000000"
check_string_result $response $expected

