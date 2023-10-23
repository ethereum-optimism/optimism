// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, OwnerManager } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { LivenessGuard } from "src/Safe/LivenessGuard.sol";
import { ISemver } from "src/universal/ISemver.sol";

// TODO(maurelian): remove me
import { console2 as console } from "forge-std/console2.sol";

/// @title LivenessModule
/// @notice This module is intended to be used in conjunction with the LivenessGuard. It should be able to
///         execute a transaction on the Safe in only a small number of cases.
contract LivenessModule is ISemver {
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

    /// @notice The address of the first owner in the linked list of owners
    address internal constant SENTINEL_OWNERS = address(0x1);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

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
            livenessGuard.lastLive(owner) < block.timestamp - livenessInterval,
            "LivenessModule: owner has signed recently"
        );

        // Calculate the new threshold
        address[] memory owners = safe.getOwners();
        uint256 numOwners = owners.length - 1;
        uint256 thresholdAfter;
        if (numOwners > minOwners) {
            // Preserves the invariant that the Safe has at least 8 owners

            thresholdAfter = get75PercentThreshold(numOwners);
            console.log("removing one owner. numOwners: %s, thresholdAfter: %s", numOwners, thresholdAfter);
            safe.execTransactionFromModule({
                to: address(safe),
                value: 0,
                data: abi.encodeCall(
                    // Call the Safe to remove the owner
                    OwnerManager.removeOwner,
                    (getPrevOwner(owner, owners), owner, thresholdAfter)
                    ),
                operation: Enum.Operation.Call
            });
        } else {
            console.log("removing all owners. numOwnersAfter: %s", numOwners);
            // The number of owners is dangerously low, so we wish to transfer the ownership of this Safe to a new
            // to the fallback owner.

            // The threshold will be 1 because we are removing all owners except the fallback owner
            thresholdAfter = 1;

            // Remove owners one at a time starting from the last owner.
            // Since we're removing them in order, the ordering will remain constant,
            //  and we shouldn't need to query the list of owners again.
            for (uint256 i = owners.length - 1; i >= 0; i--) {
                address currentOwner = owners[i];
                address prevOwner = getPrevOwner(currentOwner, owners);
                if (currentOwner != address(this)) {
                    safe.execTransactionFromModule({
                        to: address(safe),
                        value: 0,
                        data: abi.encodeCall(
                            // Call the Safe to remove the owner
                            OwnerManager.removeOwner,
                            (prevOwner, currentOwner, 1)
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
        }
        _verifyFinalState();
    }

    /// @notice A FREI-PI invariant check enforcing requirements on number of owners and threshold.
    function _verifyFinalState() internal view {
        address[] memory owners = safe.getOwners();
        uint256 numOwners = owners.length;
        require(
            (numOwners >= minOwners) || (numOwners == 1 && owners[0] == fallbackOwner),
            "LivenessModule: Safe must have the minimum number of owners, or be owned solely by the fallback owner"
        );

        // Check that the threshold is correct
        uint256 threshold = safe.getThreshold();
        require(
            threshold == get75PercentThreshold(numOwners),
            "LivenessModule: threshold must be 75% of the number of owners"
        );
    }

    /// @notice Get the previous owner in the linked list of owners
    function getPrevOwner(address owner, address[] memory owners) public pure returns (address prevOwner_) {
        for (uint256 i = 0; i < owners.length; i++) {
            if (owners[i] == owner) {
                if (i == 0) {
                    prevOwner_ = SENTINEL_OWNERS;
                    break;
                }
                prevOwner_ = owners[i - 1];
                break;
            }
        }
    }

    /// @notice For a given number of owners, return the lowest threshold which is greater than 75.
    ///         Note: this function returns 1 for numOwners == 1.
    function get75PercentThreshold(uint256 _numOwners) public pure returns (uint256 threshold_) {
        threshold_ = (_numOwners * 75 + 99) / 100;
    }
}
