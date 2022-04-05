//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title Withdrawer
 */
contract Withdrawer {
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

    /**
     * @notice Removes all ETH held in this contract from the state, by deploying a contract which
     * immediately self destructs.
     * For simplicity, this call is not incentivized as it costs very little to run.
     * Inspired by https://etherscan.io/address/0xb69fba56b2e67e7dda61c8aa057886a8d1468575#code
     */
    function burn() external {
        assembly {
            // Put this code into memory at the scratch space (first word).
            // 30 - address(this)
            // ff - selfdestruct
            mstore(0, 0x30ff)

            // Transfer all funds to a new contract that will selfdestruct
            // and destroy all the ether it holds in the process.
            pop(
                create(
                    balance(address()), // Fund the new contract with the balance of this one.
                    30, // offset
                    2 // size
                )
            )
        }
    }
}
