//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title L1Block
 * @dev This is an L2 predeploy contract that holds values from the L1
 * chain. It can only be updated by a special account that has no private
 * key managed by the L2 system. Transactions sent to this contract can
 * be thought of as "L2 system transactions".
 */
contract L1Block {
    /**
     * @notice Only the Depositor account may call setL1BlockValues().
     */
    error OnlyDepositor();

    /**
     * @notice The depositor account is a special account that sends
     * transactions to this contract.
     */
    address public constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;

    /**
     * @notice The latest L1 block number known by the L2 system
     */
    uint64 public number;

    /**
     * @notice The latest L1 timestamp known by the L2 system
     */
    uint64 public timestamp;

    /**
     * @notice The latest L1 basefee
     */
    uint256 public basefee;

    /**
     * @notice The latest L1 blockhash
     */
    bytes32 public hash;

    /**
     * @notice The number of L2 blocks in the same epoch
     */
    uint64 public sequenceNumber;

    /**
     * @notice Sets the L1 values
     * @param _number L1 blocknumber
     * @param _timestamp L1 timestamp
     * @param _basefee L1 basefee
     * @param _hash L1 blockhash
     * @param _sequenceNumber Number of L2 blocks since epoch start
     */
    function setL1BlockValues(
        uint64 _number,
        uint64 _timestamp,
        uint256 _basefee,
        bytes32 _hash,
        uint64 _sequenceNumber
    ) external {
        if (msg.sender != DEPOSITOR_ACCOUNT) {
            revert OnlyDepositor();
        }

        number = _number;
        timestamp = _timestamp;
        basefee = _basefee;
        hash = _hash;
        sequenceNumber = _sequenceNumber;
    }
}
