// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IL1BlockCGT is IProxyAdminOwnedBase {
    error L1Block_FeatureAlreadyEnabled();
    error L1Block_NotAuthorizedToSetFeature();

    event FeatureSet(bytes32 indexed feature, bool indexed enabled);

    function DEPOSITOR_ACCOUNT() external pure returns (address addr_);
    function number() external view returns (uint64);
    function timestamp() external view returns (uint64);
    function basefee() external view returns (uint256);
    function hash() external view returns (bytes32);
    function sequenceNumber() external view returns (uint64);
    function blobBaseFeeScalar() external view returns (uint32);
    function baseFeeScalar() external view returns (uint32);
    function batcherHash() external view returns (bytes32);
    function l1FeeOverhead() external view returns (uint256);
    function l1FeeScalar() external view returns (uint256);
    function blobBaseFee() external view returns (uint256);
    function operatorFeeConstant() external view returns (uint64);
    function operatorFeeScalar() external view returns (uint32);
    function daFootprintGasScalar() external view returns (uint16);
    function version() external pure returns (string memory);
    function isCustomGasToken() external view returns (bool isCustom_);
    function gasPayingTokenName() external view returns (string memory name_);
    function gasPayingTokenSymbol() external view returns (string memory symbol_);
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
    function setL1BlockValuesIsthmus() external;
    function setL1BlockValuesJovian() external;
    function gasPayingToken() external view returns (address, uint8);
    function setFeature(bytes32 _feature) external;
    function isFeatureEnabled(bytes32) external view returns (bool);

    function __constructor__() external;
}
