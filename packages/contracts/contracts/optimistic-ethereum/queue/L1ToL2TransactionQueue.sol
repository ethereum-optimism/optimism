pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { RollupQueue } from "./RollupQueue.sol";

contract L1ToL2TransactionQueue is ContractResolver, RollupQueue {
    address public l1ToL2TransactionPasser;

    constructor(
        address _addressResolver,
        address _l1ToL2TransactionPasser
    )
        public
        ContractResolver(_addressResolver)
    {
        l1ToL2TransactionPasser = _l1ToL2TransactionPasser;
    }

    function authenticateEnqueue(address _sender) public view returns (bool) {
        return _sender == l1ToL2TransactionPasser;
    }

    function authenticateDequeue(address _sender) public view returns (bool) {
        return _sender == address(resolveCanonicalTransactionChain());
    }

    function isCalldataTxQueue() public returns (bool) {
        return false;
    }


    /*
     * Contract Resolution
     */

    function resolveCanonicalTransactionChain() internal view returns (CanonicalTransactionChain) {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }
}
