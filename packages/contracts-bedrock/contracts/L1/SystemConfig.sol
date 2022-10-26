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
    address public batcher;

    event ConfigUpdate(
        uint256 indexed version,
        UpdateType indexed updateType,
        bytes data
    );

    enum UpdateType {
        BATCHER,
        GAS_CONFIG
    }

    constructor(uint256 _overhead, uint256 _scalar, address _batcher) Semver(0, 0, 1) {
        overhead = _overhead;
        scalar = _scalar;
        batcher = _batcher;
    }

    /**
     * @notice Initializer;
     */
    function initialize(address _owner) public initializer {
        __Ownable_init();
        transferOwnership(_owner);
    }

    function setBatcher(address _batcher) external onlyOwner {
        batcher = _batcher;

        bytes memory data = abi.encode(_batcher);
        emit ConfigUpdate(VERSION, UpdateType.BATCHER, data);
    }

    function setGasConfig(uint256 _overhead, uint256 _scalar) external onlyOwner {
        overhead = _overhead;
        scalar = _scalar;

        bytes memory data = abi.encode(_overhead, _scalar);
        emit ConfigUpdate(VERSION, UpdateType.GAS_CONFIG, data);
    }
}
