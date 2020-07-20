pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { ContractResolver } from "../../utils/resolvers/ContractResolver.sol";
import { ExecutionManager } from "../ExecutionManager.sol";

contract L1MessageSender is ContractResolver {
    constructor(address _addressResolver) public ContractResolver(_addressResolver) {}

    function getL1MessageSender() public returns (address) {
        ExecutionManager executionManager = resolveExecutionManager();
        return executionManager.getL1MessageSender();
    }


    /*
     * Contract Resolution
     */

    function resolveExecutionManager() internal view returns (ExecutionManager) {
        return ExecutionManager(resolveContract("ExecutionManager"));
    }
}
