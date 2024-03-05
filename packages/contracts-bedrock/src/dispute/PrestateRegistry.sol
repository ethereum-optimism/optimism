// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

/// @notice The prestate registry for a fault proof system.
/// @custom:network-specific
contract PrestateRegistry {
    /// @notice Represents a hardfork activation and the prestate for the `op-program` at that hardfork.
    /// @custom:field prestates Mapping of `VM ID` -> `Program ID` -> absolute prestate hash.
    /// @custom:field l2ActivationBlock The L2 block number at which the hardfork is activated.
    struct Hardfork {
        mapping(uint256 => mapping(uint256 => Hash)) prestates;
        uint256 l2ActivationBlock;
    }

    /// @notice Represents the prestate information for a given Fault Proof Program on a given VM.
    /// @custom:field vmID The VM ID.
    /// @custom:field programID The program ID.
    /// @custom:field prestateHash The prestate hash for the program on the given VM.
    struct PrestateInformation {
        uint256 vmID;
        uint256 programID;
        Hash prestateHash;
    }

    /// @notice The superchian configuration.
    SuperchainConfig internal immutable SUPERCHAIN_CONFIG;
    /// @notice The L2 genesis block timestamp.
    Timestamp internal immutable L2_GENESIS_BLOCK_TIMESTAMP;
    /// @notice The L2 genesis block.
    uint256 internal immutable L2_GENESIS_BLOCK;
    /// @notice The L2 chain id.
    uint256 internal immutable L2_CHAIN_ID;
    /// @notice The L2 block time.
    uint256 internal immutable L2_BLOCK_TIME;

    /// @notice A list of L2 hardforks, in ascending order. This list does not have to be exhaustive, but it must be
    ///         sorted and contain upgrades that alter the Fault Proof Program prestate hashes.
    Hardfork[] public hardforks;

    constructor(
        SuperchainConfig _superchainConfig,
        Timestamp _l2GenesisBlockTimestamp,
        uint256 _l2GenesisBlock,
        uint256 _l2ChainId,
        uint256 _l2BlockTime
    ) {
        L2_CHAIN_ID = _l2ChainId;
        L2_GENESIS_BLOCK = _l2GenesisBlock;
        L2_GENESIS_BLOCK_TIMESTAMP = _l2GenesisBlockTimestamp;
        L2_BLOCK_TIME = _l2BlockTime;
        SUPERCHAIN_CONFIG = _superchainConfig;
    }

    /// @notice Registers a hardfork with the prestate registry.
    /// @param _l2ActivationTime The L2 timestamp at which the hardfork is activated.
    /// @param _prestates The prestates for the hardfork.
    function registerHardfork(Timestamp _l2ActivationTime, PrestateInformation[] memory _prestates) external {
        // INVARIANT: Only the guardian can register hardforks.
        if (msg.sender != SUPERCHAIN_CONFIG.guardian()) revert BadAuth();

        // Convert the L2 timestamp to an L2 block number.
        uint256 activationBlock = l2TimestampToBlock(_l2ActivationTime);

        uint256 currentLength = hardforks.length;
        if (currentLength > 0) {
            Hardfork storage latestFork = hardforks[currentLength - 1];

            // INVARIANT: Hardforks must be registered in ascending order.
            if (latestFork.l2ActivationBlock >= activationBlock) revert OutOfOrderHardfork();
        }

        // Register all of the prestates for the hardfork.
        hardforks.push();
        for (uint256 i = 0; i < _prestates.length; i++) {
            PrestateInformation memory prestate = _prestates[i];
            hardforks[currentLength].prestates[prestate.vmID][prestate.programID] = prestate.prestateHash;
        }
        hardforks[currentLength].l2ActivationBlock = activationBlock;
    }

    /// @notice Revokes the latest hardfork, if it has not happened yet.
    function revokePendingFork() external {
        // INVARIANT: Only the guardian can revoke pending hardforks.
        if (msg.sender != SUPERCHAIN_CONFIG.guardian()) revert BadAuth();

        // INVARIANT: If the fork has already been activated, it cannot be revoked.
        // NOTE: This check is imperfect as we reference based on L1 time, but this drift is acceptable.
        Hardfork storage latestFork = hardforks[hardforks.length - 1];
        if (l2BlockToTimestamp(latestFork.l2ActivationBlock).raw() <= block.timestamp) {
            revert ForkAlreadyActivated();
        }

        // Remove the latest hardfork.
        hardforks.pop();
    }

    /// @notice Returns the absolute prestate in tthe latest active hardfork for a given VM and program.
    /// @param _l2BlockNumber The L2 block number at which the program is being ran until.
    /// @param _vmID The VM ID.
    /// @param _programID The program ID.
    function activePrestate(
        uint256 _l2BlockNumber,
        uint256 _vmID,
        uint256 _programID
    )
        public
        view
        returns (Hash _prestate)
    {
        uint256 currentLength = hardforks.length;

        // INVARIANT: The `hardforks` array must not be empty.
        // INVARIANT: The L2 block number must be greater than or equal to the first hardfork activation registered.
        if (hardforks.length == 0 || _l2BlockNumber < hardforks[0].l2ActivationBlock) revert NoRegisteredForks();

        // Find the hardfork via binary search.
        uint256 lo = 0;
        uint256 hi = currentLength - 1;
        while (lo <= hi) {
            uint256 mid = (lo + hi) / 2;
            if (hardforks[mid].l2ActivationBlock > _l2BlockNumber) {
                hi = mid - 1;
            } else {
                lo = mid + 1;
            }
        }

        // Grab the prestate from the hardfork definition.
        Hardfork storage fork = hardforks[hi];
        _prestate = fork.prestates[_vmID][_programID];
    }

    /// @notice Converts an L2 block number to an L2 timestamp, based off of the L2 genesis block number
    /// @param _l2BlockNumber The L2 block number
    /// @return _timestamp The L2 block number's timestamp
    function l2BlockToTimestamp(uint256 _l2BlockNumber) public view returns (Timestamp _timestamp) {
        _timestamp = Timestamp.wrap(
            L2_GENESIS_BLOCK_TIMESTAMP.raw() + uint64((_l2BlockNumber - L2_GENESIS_BLOCK) * L2_BLOCK_TIME)
        );
    }

    /// @notice Converts an L2 block number to an L2 timestamp, based off of the L2 genesis block number
    /// @param _l2Timestamp The L2 block timestamp
    /// @return l2BlockNumber_ The L2 block number
    function l2TimestampToBlock(Timestamp _l2Timestamp) public view returns (uint256 l2BlockNumber_) {
        l2BlockNumber_ = L2_GENESIS_BLOCK + ((_l2Timestamp.raw() - L2_GENESIS_BLOCK_TIMESTAMP.raw()) / L2_BLOCK_TIME);
    }
}
