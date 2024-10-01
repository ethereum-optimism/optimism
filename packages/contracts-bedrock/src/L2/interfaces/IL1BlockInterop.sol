// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

enum ConfigType {
    SET_GAS_PAYING_TOKEN,
    ADD_DEPENDENCY,
    REMOVE_DEPENDENCY
}

interface IL1BlockInterop {
    error AlreadyDependency();
    error CantRemovedDependency();
    error DependencySetSizeTooLarge();
    error NotCrossL2Inbox();
    error NotDependency();
    error NotDepositor();

    event DependencyAdded(uint256 indexed chainId);
    event DependencyRemoved(uint256 indexed chainId);
    event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol);

    function DEPOSITOR_ACCOUNT() external pure returns (address addr_);
    function baseFeeScalar() external view returns (uint32);
    function basefee() external view returns (uint256);
    function batcherHash() external view returns (bytes32);
    function blobBaseFee() external view returns (uint256);
    function blobBaseFeeScalar() external view returns (uint32);
    function dependencySetSize() external view returns (uint8);
    function depositsComplete() external;
    function gasPayingToken() external view returns (address addr_, uint8 decimals_);
    function gasPayingTokenName() external view returns (string memory name_);
    function gasPayingTokenSymbol() external view returns (string memory symbol_);
    function hash() external view returns (bytes32);
    function isCustomGasToken() external view returns (bool);
    function isDeposit() external view returns (bool isDeposit_);
    function isInDependencySet(uint256 _chainId) external view returns (bool);
    function l1FeeOverhead() external view returns (uint256);
    function l1FeeScalar() external view returns (uint256);
    function number() external view returns (uint64);
    function sequenceNumber() external view returns (uint64);
    function setConfig(ConfigType _type, bytes memory _value) external;
    function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external;
    function setL1BlockValues(
        uint64 _number,
        uint64 _timestamp,
        uint256 _basefee,
        bytes32 _hash,
        uint64 _sequenceNumber,
        bytes32 _batcherHash,
        uint256 _l1FeeOverhead,
        uint256 _l1FeeScalar
    )
        external;
    function setL1BlockValuesEcotone() external;
    function setL1BlockValuesInterop() external;
    function timestamp() external view returns (uint64);
    function version() external pure returns (string memory);

    function __constructor__() external;
}
