// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard, GuardManager } from "safe-contracts/base/GuardManager.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { GetSigners } from "src/Safe/GetSigners.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { ISemver } from "src/universal/ISemver.sol";

contract LivenessGuard is ISemver, GetSigners, BaseGuard {
    /// @notice Emitted when a new set of signers is recorded.
    /// @param signers An arrary of signer addresses.
    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    Safe public immutable safe;
    mapping(address => uint256) public lastSigned;

    constructor(Safe _safe) {
        safe = _safe;
    }

    /// @notice We just need to satisfy the BaseGuard interfae, but we don't actually need to use this method.
    function checkAfterExecution(bytes32, bool) external pure {
        return;
    }

    /// @notice Records the most recent time which any owner has signed a transaction.
    /// @dev This method is called by the Safe contract, it is critical that it does not revert, otherwise
    ///      the Safe contract will be unable to execute transactions.
    function checkTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address payable refundReceiver,
        bytes memory signatures,
        address
    )
        external
    {
        require(msg.sender == address(safe), "LivenessGuard: only Safe can call this function");

        // This call will reenter to the Safe which is calling it. This is OK because it is only reading the
        // nonce, and using the getTransactionHash() method.
        bytes32 txHash = Safe(payable(msg.sender)).getTransactionHash(
            // Transaction info
            to,
            value,
            data,
            operation,
            safeTxGas,
            // Payment info
            baseGas,
            gasPrice,
            gasToken,
            refundReceiver,
            // Signature info
            Safe(payable(msg.sender)).nonce() - 1
        );

        uint256 threshold = safe.getThreshold();
        address[] memory signers =
            _getNSigners({ dataHash: txHash, signatures: signatures, requiredSignatures: threshold });

        for (uint256 i = 0; i < signers.length; i++) {
            lastSigned[signers[i]] = block.timestamp;
        }
        emit SignersRecorded(txHash, signers);
    }

    /// @notice Enables an owner to demonstrate liveness by calling this method directly.
    ///         This is useful for owners who have not recently signed a transaction via the Safe.
    function showLiveness() external {
        require(safe.isOwner(msg.sender), "LivenessGuard: only Safe owners may demontstrate liveness");
        lastSigned[msg.sender] = block.timestamp;
        address[] memory signers = new address[](1);
        signers[0] = msg.sender;

        // todo(maurelian): Is there any need for this event to be differentiated from the one emitted in
        // checkTransaction?
        //                  Technically the 0x0 txHash does serve to identiy a call to this method.
        emit SignersRecorded(0x0, signers);
    }
}
