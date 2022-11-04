// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @title SystemConfig
 * @notice This contract is used to update L2 configuration via L1
 */
contract SystemConfig is OwnableUpgradeable, Semver {
    uint256 public constant VERSION = 0;

    uint256 public overhead;
    uint256 public scalar;
    bytes32 public batcherHash;

    event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);

    enum UpdateType {
        BATCHER,
        GAS_CONFIG
    }

    constructor(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash
    ) Semver(0, 0, 1) {
        initialize(_owner, _overhead, _scalar, _batcherHash);
    }

    /**
     * @notice Initializer;
     */
    function initialize(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash
    ) public initializer {
        __Ownable_init();
        transferOwnership(_owner);
        overhead = _overhead;
        scalar = _scalar;
        batcherHash = _batcherHash;
    }

    function setBatcherHash(bytes32 _batcherHash) external onlyOwner {
        batcherHash = _batcherHash;

        bytes memory data = abi.encode(_batcherHash);
        emit ConfigUpdate(VERSION, UpdateType.BATCHER, data);
    }

    function setGasConfig(uint256 _overhead, uint256 _scalar) external onlyOwner {
        overhead = _overhead;
        scalar = _scalar;

        bytes memory data = abi.encode(_overhead, _scalar);
        emit ConfigUpdate(VERSION, UpdateType.GAS_CONFIG, data);
    }
}
