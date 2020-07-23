pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { RollupQueue } from "./RollupQueue.sol";

contract SafetyTransactionQueue is ContractResolver, RollupQueue {
    constructor(address _addressResolver) public ContractResolver(_addressResolver) {}

    function authenticateDequeue(address _sender) public view returns (bool) {
        return _sender == address(resolveCanonicalTransactionChain());
    }


    /*
     * Contract Resolution
     */

    function resolveCanonicalTransactionChain() internal view returns (CanonicalTransactionChain) {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }
}
