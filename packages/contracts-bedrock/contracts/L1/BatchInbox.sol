// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @custom:proxied
 * @title BatchInbox
 * @notice Calldata entries of version hashes which are checked against the precompile of blobs to verify they exist
 */
// slither-disable-next-line locked-ether
contract BatchInbox is OwnableUpgradeable, Semver {
    /**
     * @custom:semver 0.0.1
     *
     * @param _owner                 The address of the owner.
     */
    constructor(
        address _owner
    ) Semver(0, 0, 1) {
        initialize(_owner);
    }

    /**
     * @notice Initializer.
     *
     * @param _owner               The address of the owner.
     */
    function initialize(
        address _owner
    ) public initializer {
        __Ownable_init();
        _transferOwnership(_owner);
    }
    /**
     * @notice appends an array of valid version hashes to the chain through calldata, each VH is checked via the VH precompile.
     * the calldata should be contingious set of 32 byte version hashes to check via precompile. Will consume memory for 1 hash and check that the a hash value was parrtoed back to indicate validity.
     *
     */
    function appendSequencerBatch() external view {
        // Revert if the provided calldata does not consist of the 4 byte selector and segments of 32 bytes.
        require((msg.data.length - 4)%32 == 0);
        // Start reading calldata after the function selector.
        uint256 cursorPosition = 4;
        // Start loop. End once there is not sufficient remaining calldata to contain a 32 byte hash.
        while(cursorPosition <= (msg.data.length - 32)) {
            assembly{
                // Allocate memory for VH
                let memPtr := mload(0x40)
                // load 32 bytes from cursorPosition in calldata to memPtr location in memory
                calldatacopy(memPtr, cursorPosition, 0x20)
                // Set free pointer before function call.
                mstore(0x40, add(memPtr, 0x20))
                let result := staticcall(1500, 0x63, memPtr, 0x20, 0, 0)
                // check the RESULT does not indicate an error.
                switch result
                // Revert if precompile RESULT indicates an error.
                case 0 { revert(0, 0) }
                // Otherwise check the RETURNDATA
                default {
                    if eq(returndatasize(), 0) {
                        revert(0, 0)
                    }
                }
            }
            cursorPosition += 32;
        }
    }
}
