//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title Withdrawor
 */
contract Withdrawor {
    uint256 public nonce;
    mapping(bytes32 => bool) public withdrawals;

    event WithdrawalInitiated(
        uint256 indexed messageNonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );

    /**
     * Initiates a withdrawal to execute on L1.
     * @param _target Address to call on L1 execution.
     * @param _gasLimit GasLimit to provide on L1.
     * @param _data Data to forward to L1 target.
     */
    function initiateWithdrawal(
        address _target,
        uint256 _gasLimit,
        bytes calldata _data
    ) external payable {
        bytes32 withdrawalHash = keccak256(
            abi.encode(nonce, msg.sender, _target, msg.value, _gasLimit, _data)
        );
        withdrawals[withdrawalHash] = true;
        nonce++;

        emit WithdrawalInitiated(nonce, msg.sender, _target, msg.value, _gasLimit, _data);
    }
}
