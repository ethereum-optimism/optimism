pragma solidity ^0.5.0;

contract RollupTransactionEvents {
    event RollupTransaction();
    event SlowQueueTransaction();
    event CanonicalTransactionChainBatch();

    function sendRollupTransaction(bytes memory _calldata) public {
        emit RollupTransaction();
    }

    function sendSlowQueueTransaction(bytes memory _calldata) public {
        emit SlowQueueTransaction();
    }

    function sendCanonicalTransactionChainBatch(bytes memory _calldata) public {
        emit CanonicalTransactionChainBatch();
    }
}
