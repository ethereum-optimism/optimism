// SPDX-License-Identifier: GPL-3.0
pragma solidity 0.8.25;

import { BasePaymaster } from "@account-abstraction/core/BasePaymaster.sol";
import { IEntryPoint } from "@account-abstraction/interfaces/IEntryPoint.sol";
import { PackedUserOperation } from "@account-abstraction/interfaces/PackedUserOperation.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { IAtomicCallRouter } from "interfaces/periphery/IAtomicCallRouter.sol";

/// @notice Opt-in gas sponsorship for approved accounts calling the atomic router.
///         The owner authorizes accounts to spend the deposit, subject to a per-operation cost cap.
///         No postOp callback or validation logs are used. This preserves the demo's fixed log prefix.
contract AtomicPaymaster is BasePaymaster {
    error AtomicPaymaster_InvalidConfiguration();
    error AtomicPaymaster_NotSponsored();

    /// @custom:semver 0.2.0
    string public constant version = "0.2.0";

    event AccountAllowed(address indexed account, bool allowed);

    mapping(address => bool) public allowedAccounts;
    address public router;
    uint256 public maxSponsoredCost;

    constructor(
        address _owner,
        address _router,
        uint256 _maxSponsoredCost
    )
        BasePaymaster(IEntryPoint(Preinstalls.EntryPoint_v070))
    {
        if (_owner == address(0) || _router == address(0) || _maxSponsoredCost == 0) {
            revert AtomicPaymaster_InvalidConfiguration();
        }
        _transferOwnership(_owner);
        router = _router;
        maxSponsoredCost = _maxSponsoredCost;
    }

    /// @notice Authorizes an account's owner to spend this paymaster's gas deposit.
    function setAccountAllowed(address _account, bool _allowed) external onlyOwner {
        allowedAccounts[_account] = _allowed;
        emit AccountAllowed(_account, _allowed);
    }

    function _validatePaymasterUserOp(
        PackedUserOperation calldata _userOp,
        bytes32,
        uint256 _maxCost
    )
        internal
        view
        override
        returns (bytes memory context_, uint256 validationData_)
    {
        if (
            !allowedAccounts[_userOp.sender] || _maxCost > maxSponsoredCost || _userOp.callData.length < 4
                || bytes4(_userOp.callData[:4]) != bytes4(keccak256("execute(address,uint256,bytes)"))
        ) revert AtomicPaymaster_NotSponsored();
        (address target, uint256 value, bytes memory data) = abi.decode(_userOp.callData[4:], (address, uint256, bytes));
        if (
            target != router || value != 0 || data.length < 4
                || (
                    bytes4(data) != IAtomicCallRouter.executeRoot.selector
                        && bytes4(data) != IAtomicCallRouter.executeRemote.selector
                        && bytes4(data) != IAtomicCallRouter.executeRootWithGas.selector
                        && bytes4(data) != IAtomicCallRouter.executeRemoteWithGas.selector
                )
        ) revert AtomicPaymaster_NotSponsored();
        return (bytes(""), 0);
    }
}
