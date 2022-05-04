//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title L1Block
 */
contract L1Block {
    /**
     * Only the Depositor account may call setL1BlockValues().
     */
    error OnlyDepositor();

    address public constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;

    uint64 public number;
    uint64 public timestamp;
    uint256 public basefee;
    bytes32 public hash;
    uint64 public sequenceNumber;

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
