// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";
import { ResourceMetering } from "../L1/ResourceMetering.sol";

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
     * @custom:value BATCHER              Represents an update to the batcher hash.
     * @custom:value GAS_CONFIG           Represents an update to txn fee config on L2.
     * @custom:value GAS_LIMIT            Represents an update to gas limit on L2.
     * @custom:value UNSAFE_BLOCK_SIGNER  Represents an update to the signer key for unsafe
     *                                    block distrubution.
     */
    enum UpdateType {
        BATCHER,
        GAS_CONFIG,
        GAS_LIMIT,
        UNSAFE_BLOCK_SIGNER
    }

    /**
     * @notice Version identifier, used for upgrades.
     */
    uint256 public constant VERSION = 0;

    /**
     * @notice Storage slot that the unsafe block signer is stored at. Storing it at this
     *         deterministic storage slot allows for decoupling the storage layout from the way
     *         that `solc` lays out storage. The `op-node` uses a storage proof to fetch this value.
     */
    bytes32 public constant UNSAFE_BLOCK_SIGNER_SLOT = keccak256("systemconfig.unsafeblocksigner");

    /**
     * @notice The amount of gas that is supplied to the system transaction.
     *         This value is value is set to 1_000_000 after the Regolith hardfork.
     */
    uint256 public immutable SYSTEM_TRANSACTION_MAX_GAS;

    int256 public immutable MAX_RESOURCE_LIMIT;

    /**
     * @notice Address of the OptimismPortal.
     */
    address public immutable PORTAL;

    /**
     * @notice Fixed L2 gas overhead, used as part of fee calculations.
     */
    uint256 public overhead;

    /**
     * @notice Dynamic L2 gas overhead, used as part of fee calculations.
     */
    uint256 public scalar;

    /**
     * @notice Identifier for the batcher. For version 1 of this configuration, this is represented
     *         as an address left-padded with zeros to 32 bytes.
     */
    bytes32 public batcherHash;

    /**
     * @notice L2 block gas limit. Can be configured and must be larger than the
     *         value returned by `minimumGasLimit()`.
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
     * @custom:semver 1.1.0
     * @notice The gas limit value should be checked offchain at deploy time to ensure that it
     *         is larger than the minimumGasLimit. This check does not happen in the constructor
     *         because all contracts are deployed first and then initialized together
     *         and calls to uninitialized contracts revert.
     *
     * @param _owner             Initial owner of the contract.
     * @param _overhead          Initial overhead value.
     * @param _scalar            Initial scalar value.
     * @param _batcherHash       Initial batcher hash.
     * @param _gasLimit          Initial gas limit.
     * @param _unsafeBlockSigner Initial unsafe block signer address.
     * @param _maxResourceLimit  Maximum amount of deposit tx gas per block.
     * @param _systemTxMaxGas    Maximum amount of gas the system tx can consume.
     */
    constructor(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash,
        uint64 _gasLimit,
        address _unsafeBlockSigner,
        int256 _maxResourceLimit,
        uint64 _systemTxMaxGas
    ) Semver(1, 1, 0) {
        MAX_RESOURCE_LIMIT = _maxResourceLimit;
        SYSTEM_TRANSACTION_MAX_GAS = _systemTxMaxGas;
        initialize(_owner, _overhead, _scalar, _batcherHash, _gasLimit, _unsafeBlockSigner);
    }

    /**
     * @notice Initializer.
     *
     * @param _owner             Initial owner of the contract.
     * @param _overhead          Initial overhead value.
     * @param _scalar            Initial scalar value.
     * @param _batcherHash       Initial batcher hash.
     * @param _gasLimit          Initial gas limit.
     * @param _unsafeBlockSigner Initial unsafe block signer address.
     */
    function initialize(
        address _owner,
        uint256 _overhead,
        uint256 _scalar,
        bytes32 _batcherHash,
        uint64 _gasLimit,
        address _unsafeBlockSigner
    ) public initializer {
        __Ownable_init();
        transferOwnership(_owner);
        overhead = _overhead;
        scalar = _scalar;
        batcherHash = _batcherHash;
        gasLimit = _gasLimit;
        _setUnsafeBlockSigner(_unsafeBlockSigner);
    }

    /**
     * @notice High level getter for the unsafe block signer address. Unsafe blocks can be
     *         propagated across the p2p network if they are signed by the key corresponding to
     *         this address.
     *
     * @return Address of the unsafe block signer.
     */
    // solhint-disable-next-line ordering
    function unsafeBlockSigner() external view returns (address) {
        address addr;
        bytes32 slot = UNSAFE_BLOCK_SIGNER_SLOT;
        assembly {
            addr := sload(slot)
        }
        return addr;
    }

    /**
     * @notice Updates the unsafe block signer address.
     *
     * @param _unsafeBlockSigner New unsafe block signer address.
     */
    function setUnsafeBlockSigner(address _unsafeBlockSigner) external onlyOwner {
        _setUnsafeBlockSigner(_unsafeBlockSigner);

        bytes memory data = abi.encode(_unsafeBlockSigner);
        emit ConfigUpdate(VERSION, UpdateType.UNSAFE_BLOCK_SIGNER, data);
    }

    /**
     * @notice Updates the batcher hash.
     *
     * @param _batcherHash New batcher hash.
     */
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
        require(_gasLimit >= minimumGasLimit(), "SystemConfig: gas limit too low");
        gasLimit = _gasLimit;

        bytes memory data = abi.encode(_gasLimit);
        emit ConfigUpdate(VERSION, UpdateType.GAS_LIMIT, data);
    }

    /**
     * @notice Low level setter for the unsafe block signer address. This function exists to
     *         deduplicate code around storing the unsafeBlockSigner address in storage.
     *
     * @param _unsafeBlockSigner New unsafeBlockSigner value.
     */
    function _setUnsafeBlockSigner(address _unsafeBlockSigner) internal {
        bytes32 slot = UNSAFE_BLOCK_SIGNER_SLOT;
        assembly {
            sstore(slot, _unsafeBlockSigner)
        }
    }

    /**
     * @notice Returns the minimum gas limit allowed by the system. If the L2
     *         gas limit is set to a value smaller than this, then it is
     *         possible for a block to be produced that uses more gas than what
     *         is allowed on L2, resulting in a liveness failure. The MAX_RESOURCE_LIMIT
     *         represents the amount of gas that can be consumed by deposits and
     *         the SYSTEM_TRANSACTION_MAX_GAS represents the maximum amount of
     *         gas the system transaction can consume in a block.
     */
    function minimumGasLimit() public view returns (uint256) {
        // we can technically not break this up into 2 variables. It
        // is a bit more explicit with 2 variables but more complex.
        return uint256(MAX_RESOURCE_LIMIT) + SYSTEM_TRANSACTION_MAX_GAS;
    }
}
