pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import {ExecutionManager} from "./ExecutionManager.sol";

/**
 * @title CreatorContract
 * @notice This contract simply deploys whatever data it is sent in the transaction calling it.
 *         It comes in handy for serving as an initial contract in rollup chains which can
 *         deploy any initial contracts.
 */
contract CreatorContract {
    address executionManagerAddress;

    constructor(address _executionManagerAddress) public {
        executionManagerAddress = _executionManagerAddress;
    }

    /**
     * @notice Fallback function which simply CREATEs a contract with whatever tx data it receives.
     */
    function() external {
        bytes4 methodId = bytes4(keccak256("ovmCREATE()") >> 224);
        address addr = executionManagerAddress;

        assembly {
            // Since this doesn't have a method ID, add 4 bytes for method ID
            let callSize := add(calldatasize, 4)

            let callBytes := mload(0x40)
            calldatacopy(add(callBytes, 4), 0, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)

            let returnData := callBytes
            let success := call(gas, addr, 0, callBytes, callSize, returnData, 500000)

            if eq(success, 0) {
                revert(0, 0)
            }

            return(returnData, returndatasize)
        }
    }
}
