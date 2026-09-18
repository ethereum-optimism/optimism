// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

// Scripts
import { Process } from "scripts/libraries/Process.sol";
import { Config } from "scripts/libraries/Config.sol";

// Testing
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Libraries
import { Claim, Duration, GameType, GameTypes, Hash, Proposal } from "src/dispute/lib/Types.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

// Interfaces
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

/// @title PastUpgrades
/// @notice Library for loading and executing past upgrades by fetching OPCM data from the
///         superchain-registry via FFI. This provides a single source of truth for past upgrade
///         configuration that can be used across ForkL1Live.s.sol and OPCM tests.
library PastUpgrades {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Dummy prestates used for testing (actual values don't matter for upgrade tests)
    bytes32 internal constant DUMMY_CANNON_PRESTATE = keccak256("CANNON");
    bytes32 internal constant DUMMY_CANNON_KONA_PRESTATE = keccak256("CANNON_KONA");
    bytes32 internal constant DUMMY_ZK_PRESTATE = keccak256("ZK");

    /// @notice Struct representing an OPCM from the registry (returned by FFI).
    ///         Note: releaseVersion is NOT the OPCM semver - query opcm.version() on-chain for that.
    struct OPCMInfo {
        address addr;
        string releaseVersion; // Contracts release version (e.g., "1.6.0"), not OPCM semver
    }

    /// @notice Struct for resolved OPCM with on-chain version from opcm.version()
    struct ResolvedOPCM {
        address addr;
        string opcmVersion; // Actual OPCM semver from opcm.version() (e.g., "6.0.0")
        SemverComp.Semver semver;
    }

    /// @notice Fetches OPCM addresses from the superchain-registry via FFI.
    /// @param _chainId The chain ID to fetch OPCMs for.
    /// @return opcms_ Array of OPCM info structs.
    function fetchOPCMs(uint256 _chainId) internal returns (OPCMInfo[] memory opcms_) {
        string[] memory command = new string[](3);
        command[0] = "scripts/go-ffi/go-ffi";
        command[1] = "opcm";
        command[2] = vm.toString(_chainId);

        bytes memory result = Process.run(command, true);

        // Handle empty result
        if (result.length == 0 || keccak256(result) == keccak256(bytes("[]"))) {
            return new OPCMInfo[](0);
        }

        // Decode the ABI-encoded array of structs
        opcms_ = abi.decode(result, (OPCMInfo[]));
    }

    /// @notice Tries to get the lastUsedOPCMVersion from SystemConfig.
    /// @param _systemConfig The SystemConfig proxy.
    /// @return version_ The version string, empty if call reverted.
    /// @return success_ True if the call succeeded, false if it reverted (pre-6.x.x chain).
    function tryGetLastUsedOPCMVersion(ISystemConfig _systemConfig)
        internal
        view
        returns (string memory version_, bool success_)
    {
        // This code is EIP-150 safe because it only runs in tests and the tests would fail if the
        // call reverts for some reason.
        // eip150-safe
        try _systemConfig.lastUsedOPCMVersion() returns (string memory v_) {
            return (v_, true);
        } catch {
            return ("", false);
        }
    }

    /// @notice Runs all past upgrades for the current chain.
    /// @param _delegateCaller The address to use as the delegate caller.
    /// @param _systemConfig The SystemConfig proxy address.
    /// @param _superchainConfig The SuperchainConfig proxy address.
    /// @param _disputeGameFactory The DisputeGameFactory (needed for V2 upgrades).
    function runPastUpgrades(
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig,
        IDisputeGameFactory _disputeGameFactory
    )
        internal
    {
        bool needsMigration = Config.devFeatureSuperRootGamesMigration()
            && !_isSuperGameType(
                IOptimismPortal2(payable(_systemConfig.optimismPortal())).anchorStateRegistry().respectedGameType()
            );

        // Fetch OPCMs from registry via FFI
        OPCMInfo[] memory opcms = fetchOPCMs(block.chainid);

        if (opcms.length == 0) {
            require(!needsMigration, "PastUpgrades: deployed v8 OPCM required");
            console.log("PastUpgrades: No OPCMs found for chain %d", block.chainid);
            return;
        }

        // V8 is the first supported historical OPCM.
        ResolvedOPCM[] memory resolved = _resolveAndFilterOPCMs(opcms);

        if (resolved.length == 0) {
            require(!needsMigration, "PastUpgrades: deployed v8 OPCM required");
            console.log("PastUpgrades: No OPCMs >= 8.x.x found for chain %d", block.chainid);
            return;
        }

        // Sort by on-chain version ascending
        _sortResolvedOPCMs(resolved);

        // Get the last used OPCM version to skip already-applied upgrades
        (string memory lastVersion, bool hasLastVersion) = tryGetLastUsedOPCMVersion(_systemConfig);

        if (!hasLastVersion) {
            console.log("PastUpgrades: SystemConfig.lastUsedOPCMVersion() reverted - chain is pre-6.x.x");
        } else {
            console.log("PastUpgrades: lastUsedOPCMVersion = %s", lastVersion);
        }

        for (uint256 i = 0; i < resolved.length; i++) {
            ResolvedOPCM memory opcm = resolved[i];

            // An applied v8 upgrade can still require the super-root migration.
            bool replayForMigration = needsMigration && opcm.semver.major == 8 && hasLastVersion
                && bytes(lastVersion).length > 0 && SemverComp.eq(opcm.opcmVersion, lastVersion)
                && opcm.addr == address(_systemConfig.lastUsedOPCM());
            if (
                hasLastVersion && bytes(lastVersion).length > 0 && SemverComp.lte(opcm.opcmVersion, lastVersion)
                    && !replayForMigration
            ) {
                console.log(
                    "PastUpgrades: Skipping OPCM %s (v%s) - already applied (lastUsed=%s)",
                    opcm.addr,
                    opcm.opcmVersion,
                    lastVersion
                );
                continue;
            }

            console.log("PastUpgrades: Running upgrade with OPCM %s (v%s)", opcm.addr, opcm.opcmVersion);

            executeV2Upgrade(opcm.addr, _delegateCaller, _systemConfig, _superchainConfig, _disputeGameFactory);
            lastVersion = opcm.opcmVersion;
            hasLastVersion = true;
            needsMigration = false;
        }
        require(!needsMigration, "PastUpgrades: deployed v8 OPCM required");
    }

    /// @notice Executes a single V2 OPCM upgrade.
    /// @param _opcm The V2 OPCM contract address.
    /// @param _delegateCaller The address to use as the delegate caller.
    /// @param _systemConfig The SystemConfig proxy address.
    /// @param _superchainConfig The SuperchainConfig proxy address.
    /// @param _disputeGameFactory The DisputeGameFactory to read init bonds from.
    function executeV2Upgrade(
        address _opcm,
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig,
        IDisputeGameFactory _disputeGameFactory
    )
        internal
    {
        {
            // Get the superchain PAO
            IProxyAdmin superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(_superchainConfig)));
            address superchainPAO = superchainProxyAdmin.owner();

            // Upgrade the SuperchainConfig first
            vm.prank(superchainPAO, true);
            (bool scSuccess,) = _opcm.delegatecall(
                abi.encodeCall(
                    IOPContractsManagerV2.upgradeSuperchain,
                    (
                        IOPContractsManagerV2.SuperchainUpgradeInput({
                            superchainConfig: _superchainConfig,
                            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
                        })
                    )
                )
            );
            // Acceptable to fail if already up to date
            scSuccess;
        }

        IAnchorStateRegistry asr = IOptimismPortal2(payable(_systemConfig.optimismPortal())).anchorStateRegistry();
        GameType respectedGameType = asr.respectedGameType();
        bool migrate = Config.devFeatureSuperRootGamesMigration() && !_isSuperGameType(respectedGameType);
        IOPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new IOPContractsManagerUtils.ExtraInstruction[](migrate ? 2 : 0);
        Proposal memory anchor;
        GameType targetGameType = respectedGameType;
        if (migrate) {
            require(SemverComp.parse(ISemver(_opcm).version()).major == 8, "PastUpgrades: deployed v8 OPCM required");
            require(
                respectedGameType.raw() == GameTypes.CANNON.raw()
                    || respectedGameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
                    || respectedGameType.raw() == GameTypes.CANNON_KONA.raw(),
                "PastUpgrades: unsupported anchor game type"
            );
            targetGameType = respectedGameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
                ? GameTypes.SUPER_PERMISSIONED
                : GameTypes.SUPER_CANNON_KONA;
            anchor = _superRootAnchor(_systemConfig, asr);
            instructions[0] = IOPContractsManagerUtils.ExtraInstruction({
                key: "overrides.cfg.startingAnchorRoot",
                data: abi.encode(anchor)
            });
            instructions[1] = IOPContractsManagerUtils.ExtraInstruction({
                key: "overrides.cfg.startingRespectedGameType",
                data: abi.encode(targetGameType)
            });
        } else {
            (anchor.root, anchor.l2SequenceNumber) = asr.getAnchorRoot();
        }

        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            _disputeGameConfigs(_disputeGameFactory, migrate);

        // Execute the V2 upgrade
        vm.prank(_delegateCaller, true);
        (bool upgradeSuccess, bytes memory reason) = _opcm.delegatecall(
            abi.encodeCall(
                IOPContractsManagerV2.upgrade,
                (
                    IOPContractsManagerV2.UpgradeInput({
                        systemConfig: _systemConfig,
                        disputeGameConfigs: disputeGameConfigs,
                        extraInstructions: instructions
                    })
                )
            )
        );
        if (!upgradeSuccess) {
            assembly {
                revert(add(reason, 0x20), mload(reason))
            }
        }
        asr = IOptimismPortal2(payable(_systemConfig.optimismPortal())).anchorStateRegistry();
        (Hash root, uint256 sequenceNumber) = asr.getAnchorRoot();
        require(
            root.raw() == anchor.root.raw() && sequenceNumber == anchor.l2SequenceNumber,
            "PastUpgrades: anchor mismatch"
        );
        require(asr.respectedGameType().raw() == targetGameType.raw(), "PastUpgrades: unexpected respected game type");
    }

    /// @notice Keeps the v8 game order and preserves registered games and their roles.
    function _disputeGameConfigs(
        IDisputeGameFactory _disputeGameFactory,
        bool _migrate
    )
        private
        view
        returns (IOPContractsManagerUtils.DisputeGameConfig[] memory configs_)
    {
        GameType[6] memory gameTypes = [
            GameTypes.CANNON,
            GameTypes.PERMISSIONED_CANNON,
            GameTypes.CANNON_KONA,
            GameTypes.SUPER_PERMISSIONED,
            GameTypes.SUPER_CANNON_KONA,
            GameTypes.ZK_DISPUTE_GAME
        ];
        configs_ = new IOPContractsManagerUtils.DisputeGameConfig[](gameTypes.length);
        for (uint256 i = 0; i < gameTypes.length; i++) {
            configs_[i].gameType = gameTypes[i];
            configs_[i].gameArgs = hex"";
            if (_migrate && i < 3) {
                continue;
            }

            GameType sourceGameType = gameTypes[i];
            if (_migrate && address(_disputeGameFactory.gameImpls(sourceGameType)) == address(0)) {
                if (sourceGameType.raw() == GameTypes.SUPER_PERMISSIONED.raw()) {
                    sourceGameType = GameTypes.PERMISSIONED_CANNON;
                } else if (sourceGameType.raw() == GameTypes.SUPER_CANNON_KONA.raw()) {
                    sourceGameType = address(_disputeGameFactory.gameImpls(GameTypes.CANNON_KONA)) != address(0)
                        ? GameTypes.CANNON_KONA
                        : GameTypes.CANNON;
                }
            }
            // Retired games must not return merely because the factory retains their bonds.
            if (address(_disputeGameFactory.gameImpls(sourceGameType)) == address(0)) {
                continue;
            }
            configs_[i].enabled = true;
            configs_[i].initBond = gameTypes[i].raw() == GameTypes.SUPER_PERMISSIONED.raw()
                ? 0
                : _disputeGameFactory.initBonds(sourceGameType);

            if (gameTypes[i].raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
                configs_[i].gameArgs = abi.encode(
                    IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                        absolutePrestate: Claim.wrap(DUMMY_CANNON_PRESTATE),
                        proposer: DisputeGames.permissionedGameProposer(_disputeGameFactory),
                        challenger: DisputeGames.permissionedGameChallenger(_disputeGameFactory)
                    })
                );
            } else if (gameTypes[i].raw() == GameTypes.SUPER_PERMISSIONED.raw()) {
                configs_[i].gameArgs = abi.encode(
                    IOPContractsManagerUtils.SuperPermissionedDisputeGameConfig({
                        proposer: DisputeGames.permissionedGameProposer(_disputeGameFactory)
                    })
                );
            } else if (gameTypes[i].raw() == GameTypes.ZK_DISPUTE_GAME.raw()) {
                LibGameArgs.ZKGameArgs memory args = LibGameArgs.decodeZK(_disputeGameFactory.gameArgs(sourceGameType));
                configs_[i].gameArgs = abi.encode(
                    IOPContractsManagerUtils.ZKDisputeGameConfig({
                        absolutePrestate: Claim.wrap(args.absolutePrestate),
                        maxChallengeDuration: Duration.wrap(args.maxChallengeDuration),
                        maxProveDuration: Duration.wrap(args.maxProveDuration),
                        challengerBond: args.challengerBond
                    })
                );
            } else {
                configs_[i].gameArgs = abi.encode(
                    IOPContractsManagerUtils.FaultDisputeGameConfig({
                        absolutePrestate: Claim.wrap(
                            gameTypes[i].raw() == GameTypes.CANNON.raw()
                                ? DUMMY_CANNON_PRESTATE
                                : DUMMY_CANNON_KONA_PRESTATE
                        )
                    })
                );
            }
        }
    }

    function _isSuperGameType(GameType _gameType) private pure returns (bool) {
        return _gameType.raw() == GameTypes.SUPER_PERMISSIONED.raw()
            || _gameType.raw() == GameTypes.SUPER_CANNON_KONA.raw() || _gameType.raw() == GameTypes.ZK_DISPUTE_GAME.raw();
    }

    /// @notice Commits the live anchor output root at its canonical L2 timestamp.
    function _superRootAnchor(
        ISystemConfig _systemConfig,
        IAnchorStateRegistry _asr
    )
        private
        view
        returns (Proposal memory)
    {
        string memory chain = vm.readFile(
            string.concat(
                "../../superchain-registry/superchain/configs/",
                Config.forkBaseChain(),
                "/",
                Config.forkOpChain(),
                ".toml"
            )
        );
        uint256 chainId = vm.parseTomlUint(chain, ".chain_id");
        require(chainId == _systemConfig.l2ChainId(), "PastUpgrades: registry chain mismatch");
        (Hash outputRoot, uint256 blockNumber) = _asr.getAnchorRoot();
        require(outputRoot.raw() != bytes32(0), "PastUpgrades: empty output-root anchor");
        uint256 timestamp = vm.parseTomlUint(chain, ".genesis.l2_time")
            + (blockNumber - vm.parseTomlUint(chain, ".genesis.l2.number")) * vm.parseTomlUint(chain, ".block_time");
        require(timestamp < type(uint64).max && timestamp <= block.timestamp, "PastUpgrades: invalid anchor timestamp");
        Types.OutputRootWithChainId[] memory roots = new Types.OutputRootWithChainId[](1);
        roots[0] = Types.OutputRootWithChainId({ chainId: chainId, root: outputRoot.raw() });
        return Proposal({
            root: Hash.wrap(
                Hashing.hashSuperRootProof(
                    Types.SuperRootProof({ version: 0x01, timestamp: uint64(timestamp), outputRoots: roots })
                )
            ),
            l2SequenceNumber: timestamp
        });
    }

    /// @notice Resolves on-chain OPCM versions and filters out versions below v8.
    /// @param _opcms The OPCMs from FFI
    /// @return resolved_ The resolved and filtered OPCMs
    function _resolveAndFilterOPCMs(OPCMInfo[] memory _opcms) private view returns (ResolvedOPCM[] memory resolved_) {
        // First pass: count valid OPCMs
        uint256 count = 0;
        for (uint256 i = 0; i < _opcms.length; i++) {
            string memory opcmVersion = ISemver(_opcms[i].addr).version();
            SemverComp.Semver memory sv = SemverComp.parse(opcmVersion);
            if (sv.major >= 8) {
                count++;
            }
        }

        // Second pass: populate array
        resolved_ = new ResolvedOPCM[](count);
        uint256 idx = 0;
        for (uint256 i = 0; i < _opcms.length; i++) {
            string memory opcmVersion = ISemver(_opcms[i].addr).version();
            SemverComp.Semver memory sv = SemverComp.parse(opcmVersion);
            if (sv.major >= 8) {
                resolved_[idx] = ResolvedOPCM({ addr: _opcms[i].addr, opcmVersion: opcmVersion, semver: sv });
                idx++;
            }
        }
    }

    /// @notice Sorts resolved OPCMs by semver ascending (bubble sort)
    /// @param _resolved The array to sort in-place
    function _sortResolvedOPCMs(ResolvedOPCM[] memory _resolved) private pure {
        uint256 n = _resolved.length;
        for (uint256 i = 0; i < n; i++) {
            for (uint256 j = i + 1; j < n; j++) {
                if (SemverComp.lt(_resolved[j].opcmVersion, _resolved[i].opcmVersion)) {
                    ResolvedOPCM memory temp = _resolved[i];
                    _resolved[i] = _resolved[j];
                    _resolved[j] = temp;
                }
            }
        }
    }
}
