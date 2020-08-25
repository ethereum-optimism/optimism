pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { RollupQueue } from "../optimistic-ethereum/queue/RollupQueue.sol";

/**
 * @title RollupQueueImplementation
 */
contract RollupQueueImplementation is RollupQueue {
    /*
     * Public Functions
     */
    
    function enqueueTx(
        bytes memory _tx
    ) public {
        _enqueue(_tx);
    }

    function dequeue() public {
        _dequeue();
    }

}
