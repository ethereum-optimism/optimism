pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { AddressResolver } from "../../utils/resolvers/AddressResolver.sol";
import { ExecutionManager } from "../ExecutionManager.sol";

contract L1MessageSender {
    AddressResolver public addressResolver;

    constructor(
        address _addressResolver
    ) public {
        addressResolver = AddressResolver(_addressResolver);
    }

    function getL1MessageSender() public returns (address) {
        return executionManager().getL1MessageSender();
    }


    /*
     * Address Resolution
     */

    function executionManager() internal view returns (ExecutionManager) {
        return ExecutionManager(addressResolver.getAddress("ExecutionManager"));
    }
}
