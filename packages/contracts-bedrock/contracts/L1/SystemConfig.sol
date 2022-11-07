// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @title SystemConfig
 * @notice The SystemConfig contract is used to manage configuration of an Optimism network. All
 *         configuration is stored on L1 and picked up by L2 as part of the derviation of the L2
 *         chain.
 */
contract SystemConfig is OwnableUpgradeable, Semver {
    /**
     * @notice Enum representing different types of updates.
     *
     * @custom:value BATCHER    Represents an update to the batcher hash.
     * @custom:value GAS_CONFIG Represents an update to txn fee config on L2.
     * @custom:value GAS_LIMIT  Represents an update to gas limit on L2.
     */
    enum UpdateType {
        BATCHER,
        GAS_CONFIG,
        GAS_LIMIT
    }

    /**
     * @notice Version identifier, used for upgrades.
     */
    uint256 public constant VERSION = 0;

    /**
     * @notice Minimum gas limit. This should not be lower than the maximum deposit
     *         gas resource limit in the ResourceMetering contract used by OptimismPortal,
     *         to ensure the L2 block always has sufficient gas to process deposits.
     */
    uint64 public constant MINIMUM_GAS_LIMIT = 8_000_000;

    /**
     * @notice Fixed L2 gas overhead.
     */
    uint256 public overhead;

    /**
     * @notice Dynamic L2 gas overhead.
     */
    uint256 public scalar;

    /**
     * @notice Identifier for the batcher. For version 1 of this configuration, this is represented
     *         as an address left-padded with zeros to 32 bytes.
     */
    bytes32 public batcherHash;

    /**
     * @notice L2 gas limit.
     */
    uint64 public gasLimit;

    /**
     * @notice Emitted when configuration is updated
     *
     * @param version    SystemConfig version.
     * @param updateType Type of update.
     * @param data       Encoded update data.
     */
    event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);

    /**
     * @custom:semver 0.0.1
     *
     * @param _owner       Initial owner of the contract.
     * @param _overhead    Initial overhead value.
     * @param _scalar      Initial scalar value.
     * @param _batcherHash Initial batcher hash.
     * @param _gasLimit    Initial gas limit.
     */
    constructor(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash,
        uint64 _gasLimit
    ) Semver(0, 0, 1) {
        initialize(_owner, _overhead, _scalar, _batcherHash, _gasLimit);
    }

    /**
     * @notice Initializer.
     *
     * @param _owner       Initial owner of the contract.
     * @param _overhead    Initial overhead value.
     * @param _scalar      Initial scalar value.
     * @param _batcherHash Initial batcher hash.
     * @param _gasLimit    Initial gas limit.
     */
    function initialize(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash,
        uint64 _gasLimit
    ) public initializer {
        require(_gasLimit >= MINIMUM_GAS_LIMIT, "SystemConfig: gas limit too low");
        __Ownable_init();
        transferOwnership(_owner);
        overhead = _overhead;
        scalar = _scalar;
        batcherHash = _batcherHash;
        gasLimit = _gasLimit;
    }

    /**
     * @notice Updates the batcher hash.
     *
     * @param _batcherHash New batcher hash.
     */
    // solhint-disable-next-line ordering
    function setBatcherHash(bytes32 _batcherHash) external onlyOwner {
        batcherHash = _batcherHash;

        bytes memory data = abi.encode(_batcherHash);
        emit ConfigUpdate(VERSION, UpdateType.BATCHER, data);
    }

    /**
     * @notice Updates gas config.
     *
     * @param _overhead New overhead value.
     * @param _scalar   New scalar value.
     */
    function setGasConfig(uint256 _overhead, uint256 _scalar) external onlyOwner {
        overhead = _overhead;
        scalar = _scalar;

        bytes memory data = abi.encode(_overhead, _scalar);
        emit ConfigUpdate(VERSION, UpdateType.GAS_CONFIG, data);
    }

    /**
     * @notice Updates the L2 gas limit.
     *
     * @param _gasLimit New gas limit.
     */
    function setGasLimit(uint64 _gasLimit) external onlyOwner {
        require(_gasLimit >= MINIMUM_GAS_LIMIT, "SystemConfig: gas limit too low");
        gasLimit = _gasLimit;

        bytes memory data = abi.encode(_gasLimit);
        emit ConfigUpdate(VERSION, UpdateType.GAS_LIMIT, data);
    }
}
