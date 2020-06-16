pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract L1ToL2TransactionEvents {
    event L1ToL2Transaction();
    event SlowQueueTransaction();
    event CanonicalTransactionChainBatch();

    function sendL1ToL2Transaction(bytes memory _calldata) public {
        emit L1ToL2Transaction();
    }

    function sendSlowQueueTransaction(bytes memory _calldata) public {
        emit SlowQueueTransaction();
    }

    function sendCanonicalTransactionChainBatch(bytes memory _calldata) public {
        emit CanonicalTransactionChainBatch();
    }
}
