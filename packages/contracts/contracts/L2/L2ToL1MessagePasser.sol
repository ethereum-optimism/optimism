// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { WithdrawalVerifier } from "../libraries/Lib_WithdrawalVerifier.sol";
import { Burner } from "./Burner.sol";

/**
 * @title L2ToL1MessagePasser
 * TODO: should this be renamed to L2OptimismPortal?
 */
contract L2ToL1MessagePasser {
    /**********
     * Events *
     **********/

    /**
     * @notice Emitted any time a withdrawal is initiated.
     * @param nonce Unique value corresponding to each withdrawal.
     * @param sender The L2 account address which initiated the withdrawal.
     * @param target The L1 account address the call will be send to.
     * @param value The ETH value submitted for withdrawal, to be forwarded to the target.
     * @param gasLimit The minimum amount of gas that must be provided when withdrawing on L1.
     * @param data The data to be forwarded to the target on L1.
     */
    event WithdrawalInitiated(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );

    /**
     * @notice Emitted when the balance of this contract is burned.
     */
    event WithdrawerBalanceBurnt(uint256 indexed amount);

    /*************
     * Variables *
     *************/

    /**
     * @notice Includes the message hashes for all withdrawals
     */
    mapping(bytes32 => bool) public sentMessages;

    /**
     * @notice A unique value hashed with each withdrawal.
     */
    uint256 public nonce;

    /********************
     * Public Functions *
     ********************/

    /**
     * @notice Allow users to withdraw by sending ETH
     * directly to this contract.
     * TODO: maybe this should be only EOA
     */
    receive() external payable {
        initiateWithdrawal(msg.sender, 100000, bytes(""));
    }

    /**
     * @notice Initiates a withdrawal to execute on L1.
     * TODO: message hashes must be migrated since the legacy
     * hashes are computed differently
     * @param _target Address to call on L1 execution.
     * @param _gasLimit GasLimit to provide on L1.
     * @param _data Data to forward to L1 target.
     */
    function initiateWithdrawal(
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    ) public payable {
        bytes32 withdrawalHash = WithdrawalVerifier.withdrawalHash(
            nonce,
            msg.sender,
            _target,
            msg.value,
            _gasLimit,
            _data
        );

        sentMessages[withdrawalHash] = true;

        emit WithdrawalInitiated(nonce, msg.sender, _target, msg.value, _gasLimit, _data);
        unchecked {
            ++nonce;
        }
    }

    /**
     * @notice Removes all ETH held in this contract from the state, by deploying a contract which
     * immediately self destructs.
     * For simplicity, this call is not incentivized as it costs very little to run.
     * Inspired by https://etherscan.io/address/0xb69fba56b2e67e7dda61c8aa057886a8d1468575#code
     */
    function burn() external {
        uint256 balance = address(this).balance;
        new Burner{ value: balance }();
        emit WithdrawerBalanceBurnt(balance);
    }
}
