//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { DepositFeed } from "./DepositFeed.sol";
import { WithdrawalVerifier } from "./Lib_WithdrawalVerifier.sol";
import { L2OutputOracle } from "./L2OutputOracle.sol";

contract OptimismPortal is DepositFeed {
    event WithdrawalFinalized(bytes32 indexed);

    // Value used to reset the l2Sender. This is more gas efficient that setting it to zero.
    address internal constant DEFAULT_L2_SENDER = 0x000000000000000000000000000000000000dEaD;

    uint256 public immutable FINALIZATION_WINDOW;
    L2OutputOracle public immutable L2_ORACLE;

    address public l2Sender = DEFAULT_L2_SENDER;
    mapping(bytes32 => bool) public finalizedWithdrawals;

    constructor(L2OutputOracle _l2Oracle, uint256 _finalizationWindow) {
        L2_ORACLE = _l2Oracle;
        FINALIZATION_WINDOW = _finalizationWindow;
    }

    /**
     * Accepts value so that users can send ETH directly to this contract and
     * have the funds be deposited to their address on L2.
     * Note: this is intended as a convenience function for EOAs. Contracts should call the
     * depositTransaction() function directly.
     */
    receive() external payable {
        depositTransaction(msg.sender, msg.value, 30000, false, bytes(""));
    }

    /**
     * Finalizes a withdrawal transaction.
     * @param _nonce Nonce for the provided message.
     * @param _sender Message sender address on L2.
     * @param _target Target address on L1.
     * @param _data Data to send to the target.
     * @param _gasLimit Gas to be forwarded to the target.
     * @param _outputRootProof Inclusion proof of the withdrawer contracts storage root.
     * @param _withdrawalProof Inclusion proof for the given withdrawal in the withdrawer contract.
     */
    function finalizeWithdrawalTransaction(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data,
        WithdrawalVerifier.OutputRootProof calldata _outputRootProof,
        bytes calldata _withdrawalProof
    ) external {
        // Check that the timestamp is 7 days old.
        require(_outputRootProof.timestamp <= block.timestamp - FINALIZATION_WINDOW, "Too soon");

        // Get the output root and verify that the withdrawer contract's storage root is contained
        // in it.
        bytes32 outputRoot = L2_ORACLE.getL2Output(_outputRootProof.timestamp);
        require(
            WithdrawalVerifier._verifyWithdrawerStorageRoot(outputRoot, _outputRootProof) == true,
            "Calculated output root does not match expected value"
        );

        // Verify that the hash of the withdrawal transaction's arguments are included in the
        // storage hash of the withdrawer contract.
        bytes32 withdrawalHash = keccak256(
            abi.encode(_nonce, _sender, _target, _value, _gasLimit, _data)
        );
        require(
            WithdrawalVerifier._verifyWithdrawalInclusion(
                withdrawalHash,
                _outputRootProof.withdrawerStorageRoot,
                _withdrawalProof
            ) == true,
            "Withdrawal transaction not found in storage"
        );

        // Check that this withdrawal has not already been finalized.
        require(finalizedWithdrawals[withdrawalHash] == false, "Withdrawal already finalized");

        // Make the call.
        l2Sender = _sender;
        _target.call{ value: _value, gas: _gasLimit }(_data);
        l2Sender = DEFAULT_L2_SENDER;

        // All withdrawals are immediately finalized. If the ability to replay a transaction is
        // required, that support can be provided in external contracts.
        finalizedWithdrawals[withdrawalHash] = true;
        emit WithdrawalFinalized(withdrawalHash);
    }
}
