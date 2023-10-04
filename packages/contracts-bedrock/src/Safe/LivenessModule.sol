// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

/// @title LivenessModule
/// @notice This module is intended to be used in conjunction with the LivenessGuard. It should be able to
///         execute a transaction on the Safe in only a small number of cases:
contract LivenessModule {
    /// @notice The Safe contract instance
    Safe public safe;

    /// @notice The LivenessGuard contract instance
    LivenessGuard public livenessGuard;

    /// @notice The interval, in seconds, during which an owner must have demonstrated liveness
    uint256 public livenessInterval;

    /// @notice The minimum number of owners before ownership of the safe is transferred to the fallback owner.
    uint256 public minOwners;

    /// @notice The fallback owner of the Safe
    address public fallbackOwner;

    // Constructor to initialize the Safe and baseModule instances
    constructor(
        Safe _safe,
        LivenessGuard _livenessGuard,
        uint256 _livenessInterval,
        uint256 _minOwners,
        address _fallbackOwner
    ) {
        safe = _safe;
        livenessGuard = _livenessGuard;
        livenessInterval = _livenessInterval;
        minOwners = _minOwners;
        fallbackOwner = _fallbackOwner;
    }

    /// @notice This function can be called by anyone to remove an owner that has not signed a transaction
    ///         during the livness interval. If the number of owners drops below
    function removeOwner(address owner) external {
        // Check that the owner has not signed a transaction in the last 30 days
        require(
            livenessGuard.lastSigned(owner) < block.timestamp - livenessInterval,
            "LivenessModule: owner has signed recently"
        );

        // Calculate the new threshold
        uint256 numOwnersAfter = safe.getOwners().length - 1;
        uint256 thresholdAfter = get75PercentThreshold(numOwnersAfter);
        if (numOwnersAfter >= 8) {
            safe.execTransactionFromModule({
                to: address(safe),
                value: 0,
                data: abi.encodeCall(
                    // Call the Safe to remove the owner
                    OwnerManager.removeOwner,
                    (getPrevOwner(owner), owner, thresholdAfter)
                    ),
                operation: Enum.Operation.Call
            });
        } else {
            // The number of owners is dangerously low, so we wish to transfer the ownership of this Safe to a new
            // to the fallback owner.
            // Remove owners one at a time starting from the last owner.
            // Since we're removing them in order, the ordering will remain constant,
            //  and we shouldn't need to query the list of owners again.
            address[] memory owners = safe.getOwners();
            for (uint256 i = owners.length - 1; i >= 0; i--) {
                address currentOwner = owners[i];
                if (currentOwner != address(this)) {
                    safe.execTransactionFromModule({
                        to: address(safe),
                        value: 0,
                        data: abi.encodeCall(
                            // Call the Safe to remove the owner
                            OwnerManager.removeOwner,
                            (
                                getPrevOwner(currentOwner),
                                currentOwner,
                                1 // The threshold is 1 because we are removing all owners except the fallback owner
                            )
                            ),
                        operation: Enum.Operation.Call
                    });
                }
            }

            // Add the fallback owner as the sole owner of the Safe
            safe.execTransactionFromModule({
                to: address(safe),
                value: 0,
                data: abi.encodeCall(OwnerManager.addOwnerWithThreshold, (fallbackOwner, 1)),
                operation: Enum.Operation.Call
            });

            address[] memory ownersAfter = safe.getOwners();
            require(
                ownersAfter.length == 1 && ownersAfter[0] == fallbackOwner,
                "LivenessModule: fallback owner was not added as the sole owner"
            );
        }
    }

    /// @notice Get the previous owner in the linked list of owners
    function getPrevOwner(address owner) public view returns (address prevOwner_) {
        address[] memory owners = safe.getOwners();
        prevOwner_ = address(0);
        for (uint256 i = 0; i < owners.length; i++) {
            if (owners[i] == owner) {
                prevOwner_ = owners[i - 1];
                break;
            }
        }
    }

    /// @notice For a given number of owners, return the lowest threshold which is greater than 75.
    function get75PercentThreshold(uint256 _numOwners) public view returns (uint256 threshold_) {
        threshold_ = (_numOwners * 75 + 99) / 100;
    }
}
