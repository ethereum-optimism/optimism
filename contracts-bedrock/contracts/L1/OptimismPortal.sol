//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle } from "./L2OutputOracle.sol";
import { WithdrawalVerifier } from "../libraries/Lib_WithdrawalVerifier.sol";
import { AddressAliasHelper } from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";
import { ExcessivelySafeCall } from "../libraries/ExcessivelySafeCall.sol";

/**
 * @title OptimismPortal
 * This contract should be deployed behind an upgradable proxy.
 */
contract OptimismPortal {
    /**********
     * Errors *
     **********/

    /**
     * @notice Error emitted when the output root proof is invalid.
     */
    error InvalidOutputRootProof();

    /**
     * @notice Error emitted when the withdrawal inclusion proof is invalid.
     */
    error InvalidWithdrawalInclusionProof();

    /**
     * @notice Error emitted when a withdrawal has already been finalized.
     */
    error WithdrawalAlreadyFinalized();

    /**
     * @notice Error emitted on deposits which create a new contract with a non-zero target.
     */
    error NonZeroCreationTarget();

    /**********
     * Events *
     **********/

    /**
     * @notice Emitted when a Transaction is deposited from L1 to L2. The parameters of this
     * event are read by the rollup node and used to derive deposit transactions on L2.
     */
    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 mint,
        uint256 value,
        uint64 gasLimit,
        bool isCreation,
        bytes data
    );

    /**
     * @notice Emitted when a withdrawal is finalized
     */
    event WithdrawalFinalized(bytes32 indexed, bool success);

    /*************
     * Constants *
     *************/

    /**
     * @notice Value used to reset the l2Sender, this is more efficient than setting it to zero.
     */
    address internal constant DEFAULT_L2_SENDER = 0x000000000000000000000000000000000000dEaD;

    /*************
     * Variables *
     *************/

    /**
     * @notice Minimum time that must elapse before a withdrawal can be finalized.
     */
    uint256 public immutable FINALIZATION_PERIOD;

    /**
     * @notice Address of the L2OutputOracle.
     */
    L2OutputOracle public immutable L2_ORACLE;

    /**
     * @notice Public variable which can be used to read the address of the L2 account which
     * initated the withdrawal. Can also be used to determine whether or not execution is occuring
     * downstream of a call to finalizeWithdrawalTransaction().
     */
    address public l2Sender = DEFAULT_L2_SENDER;

    /**
     * @notice A list of withdrawal hashes which have been successfully finalized.
     * Used for replay protection.
     */
    mapping(bytes32 => bool) public finalizedWithdrawals;

    /***************
     * Constructor *
     ***************/

    constructor(L2OutputOracle _l2Oracle, uint256 _finalizationPeriod) {
        L2_ORACLE = _l2Oracle;
        FINALIZATION_PERIOD = _finalizationPeriod;
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * @notice Accepts value so that users can send ETH directly to this contract and
     * have the funds be deposited to their address on L2.
     * @dev This is intended as a convenience function for EOAs. Contracts should call the
     * depositTransaction() function directly.
     */
    receive() external payable {
        depositTransaction(msg.sender, msg.value, 100000, false, bytes(""));
    }

    /**
     * @notice Accepts deposits of ETH and data, and emits a TransactionDeposited event for use in
     * deriving deposit transactions.
     * @param _to The L2 destination address.
     * @param _value The ETH value to send in the deposit transaction.
     * @param _gasLimit The L2 gasLimit.
     * @param _isCreation Whether or not the transaction should be contract creation.
     * @param _data The input data.
     */
    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    ) public payable {
        // Differentiate between sending to address(0)
        // and creating a contract
        if (_isCreation && _to != address(0)) {
            revert NonZeroCreationTarget();
        }

        address from = msg.sender;
        // Transform the from-address to its alias if the caller is a contract.
        if (msg.sender != tx.origin) {
            from = AddressAliasHelper.applyL1ToL2Alias(msg.sender);
        }

        emit TransactionDeposited(from, _to, msg.value, _value, _gasLimit, _isCreation, _data);
    }

    /**
     * @notice Finalizes a withdrawal transaction.
     * @param _nonce Nonce for the provided message.
     * @param _sender Message sender address on L2.
     * @param _target Target address on L1.
     * @param _value ETH to send to the target.
     * @param _gasLimit Gas to be forwarded to the target.
     * @param _data Data to send to the target.
     * @param _l2Timestamp L2 timestamp of the outputRoot.
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
        uint256 _l2Timestamp,
        WithdrawalVerifier.OutputRootProof calldata _outputRootProof,
        bytes calldata _withdrawalProof
    ) external payable {
        // Prevent reentrency
        require(_target != address(this), "Cannot send message to self.");

        // Get the output root.
        L2OutputOracle.OutputProposal memory proposal = L2_ORACLE.getL2Output(_l2Timestamp);

        // Ensure that enough time has passed since the proposal was submitted
        // before allowing a withdrawal. A fault proof should be submitted
        // before this check is allowed to pass.
        require(
            block.timestamp > proposal.timestamp + FINALIZATION_PERIOD,
            "Proposal is not yet finalized."
        );

        // Verify that the output root can be generated with the elements in the proof.
        if (proposal.outputRoot != WithdrawalVerifier._deriveOutputRoot(_outputRootProof)) {
            revert InvalidOutputRootProof();
        }

        // Verify that the hash of the withdrawal transaction's arguments are included in the
        // storage hash of the withdrawer contract.
        bytes32 withdrawalHash = WithdrawalVerifier.withdrawalHash(
            _nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );

        // Verify proof that a withdrawal on L2 was initated
        if (
            WithdrawalVerifier._verifyWithdrawalInclusion(
                withdrawalHash,
                _outputRootProof.withdrawerStorageRoot,
                _withdrawalProof
            ) == false
        ) {
            revert InvalidWithdrawalInclusionProof();
        }

        // Check that this withdrawal has not already been finalized.
        if (finalizedWithdrawals[withdrawalHash] == true) {
            revert WithdrawalAlreadyFinalized();
        }

        // Set the withdrawal as finalized
        finalizedWithdrawals[withdrawalHash] = true;

        // Save enough gas so that the call cannot use up all of the gas
        require(gasleft() >= _gasLimit + 20000, "Insufficient gas to finalize withdrawal.");

        // Set the l2Sender so that other contracts can know which account
        // on L2 is making the withdrawal
        l2Sender = _sender;
        // Make the call and ensure that a contract cannot out of gas
        // us by returning a huge amount of data
        (bool success, ) = ExcessivelySafeCall.excessivelySafeCall(
            _target,
            _gasLimit,
            _value,
            0,
            _data
        );
        // Be sure to reset the l2Sender
        l2Sender = DEFAULT_L2_SENDER;

        // All withdrawals are immediately finalized. Replayability can
        // be achieved through contracts built on top of this contract
        emit WithdrawalFinalized(withdrawalHash, success);
    }
}
