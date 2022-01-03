//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title L1Block
 */
contract L1Block {
    address public constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;

    uint256 public number;
    uint256 public timestamp;
    uint256 public basefee;
    bytes32 public hash;

    function setL1BlockValues(
        uint256 _number,
        uint256 _timestamp,
        uint256 _basefee,
        bytes32 _hash
    ) external {
        require(
            msg.sender == DEPOSITOR_ACCOUNT,
            "Only callable by L1 Attributes Depositor Account"
        );

        number = _number;
        timestamp = _timestamp;
        basefee = _basefee;
        hash = _hash;
    }
}
