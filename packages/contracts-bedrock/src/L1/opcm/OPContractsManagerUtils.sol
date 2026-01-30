// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Blueprint } from "src/libraries/Blueprint.sol";
import { Constants } from "src/libraries/Constants.sol";
import { GameType, GameTypes } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";

/// @title OPContractsManagerUtils
/// @notice OPContractsManagerUtils is a contract that provides utility functions for the OPContractsManager.
contract OPContractsManagerUtils {
    /// @notice Helper struct for deploying proxies, keeps code cleaner.
    struct ProxyDeployArgs {
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        uint256 l2ChainId;
        string saltMixer;
    }

    /// @notice Struct that represents an additional instruction for an upgrade. Each upgrade has
    ///         its own set of extra upgrade instructions that may or may not be required. We use
    ///         this struct to keep the upgrade interface the same each time.
    struct ExtraInstruction {
        string key;
        bytes data;
    }

    /// @notice Emitted when a proxy is created by this contract.
    /// @param name  The name of the proxy.
    /// @param proxy The address of the proxy.
    event ProxyCreation(string name, address proxy);

    /// @notice Thrown when user attempts to downgrade a contract.
    /// @param _contract The address of the contract that was attempted to be downgraded.
    error OPContractsManagerUtils_DowngradeNotAllowed(address _contract);

    /// @notice Thrown when user attempts to deploy a contract with extra version tags in production.
    /// @param _contract The address of the contract with extra version tags.
    error OPContractsManagerUtils_ExtraTagInProd(address _contract);

    /// @notice Thrown when a config load fails.
    /// @param _name The name of the config that failed to load.
    error OPContractsManagerUtils_ConfigLoadFailed(string _name);

    /// @notice Thrown when a proxy must be loaded but couldn't be.
    /// @param _name The name of the proxy that couldn't be loaded.
    error OPContractsManagerUtils_ProxyMustLoad(string _name);

    /// @notice Container of blueprint and implementation contract addresses.
    IOPContractsManagerContainer public immutable contractsContainer;

    /// @param _contractsContainer The container of blueprint and implementation contract addresses.
    constructor(IOPContractsManagerContainer _contractsContainer) {
        contractsContainer = _contractsContainer;
    }

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    ///         configuration's convention. This convention is
    ///         `versionByte || keccak256(bytes32(chainId))[:19]`, where || denotes concatenation,
    ///         versionByte is 0x00, and chainId is a uint256.
    ///         https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    /// @param _l2ChainId The L2 chain ID to map to an L1 batch inbox address.
    /// @return Chain ID mapped to an L1 batch inbox address.
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) external pure returns (address) {
        bytes1 versionByte = 0x00;
        bytes32 hashedChainId = keccak256(bytes.concat(bytes32(_l2ChainId)));
        bytes19 first19Bytes = bytes19(hashedChainId);
        return address(uint160(bytes20(bytes.concat(versionByte, first19Bytes))));
    }

    /// @notice Computes a unique salt for a contract deployment.
    /// @param _l2ChainId The L2 chain ID of the chain being deployed to.
    /// @param _saltMixer The salt mixer to use for the deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @return The computed salt.
    function computeSalt(
        uint256 _l2ChainId,
        string memory _saltMixer,
        string memory _contractName
    )
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(_l2ChainId, _saltMixer, _contractName));
    }

    /// @notice Helper function to check if an instruction matches a given key.
    /// @param _instruction The instruction to check.
    /// @param _key The key of the instruction to check for.
    /// @return True if the instruction matches, false otherwise.
    function isMatchingInstructionByKey(
        ExtraInstruction memory _instruction,
        string memory _key
    )
        public
        pure
        returns (bool)
    {
        return LibString.eq(_instruction.key, _key);
    }

    /// @notice Helper function to check if an instruction matches a given key and data.
    /// @param _instruction The instruction to check.
    /// @param _key The key of the instruction to check for.
    /// @param _data The data of the instruction to check for.
    /// @return True if the instruction matches, false otherwise.
    function isMatchingInstruction(
        ExtraInstruction memory _instruction,
        string memory _key,
        bytes memory _data
    )
        public
        pure
        returns (bool)
    {
        return LibString.eq(_instruction.key, _key) && LibString.eq(string(_instruction.data), string(_data));
    }

    /// @notice Helper function to check if a given instruction is present in a list of extra
    ///         upgrade instructions.
    /// @param _instructions The list of extra upgrade instructions.
    /// @param _key The key of the instruction to check for.
    /// @param _data The data of the instruction to check for.
    /// @return True if the instruction is present, false otherwise.
    function hasInstruction(
        ExtraInstruction[] memory _instructions,
        string memory _key,
        bytes memory _data
    )
        public
        pure
        returns (bool)
    {
        for (uint256 i = 0; i < _instructions.length; i++) {
            if (isMatchingInstruction(_instructions[i], _key, _data)) {
                return true;
            }
        }
        return false;
    }

    /// @notice Helper function to get an instruction by key.
    /// @param _instructions The list of extra upgrade instructions.
    /// @param _key The key of the instruction to get.
    /// @return The instruction, or an empty instruction if the instruction is not found.
    function getInstructionByKey(
        ExtraInstruction[] memory _instructions,
        string memory _key
    )
        public
        pure
        returns (ExtraInstruction memory)
    {
        for (uint256 i = 0; i < _instructions.length; i++) {
            if (LibString.eq(_instructions[i].key, _key)) {
                return _instructions[i];
            }
        }
        return ExtraInstruction({ key: "", data: bytes("") });
    }

    /// @notice Helper function to load data from a source contract as bytes.
    /// @param _source The source contract to load the data from.
    /// @param _selector The selector of the function to call on the source contract.
    /// @param _name The name of the field to load.
    /// @param _instructions The extra upgrade instructions for the data load.
    /// @return Data retrieved from the source contract.
    function loadBytes(
        address _source,
        bytes4 _selector,
        string memory _name,
        ExtraInstruction[] memory _instructions
    )
        external
        view
        returns (bytes memory)
    {
        // If an override exists for this load, return the override data.
        ExtraInstruction memory overrideInstruction = getInstructionByKey(_instructions, _name);
        if (bytes(overrideInstruction.key).length > 0) {
            return overrideInstruction.data;
        }

        // Otherwise, load the data from the source contract.
        (bool success, bytes memory result) = address(_source).staticcall(abi.encodePacked(_selector));
        if (!success) {
            revert OPContractsManagerUtils_ConfigLoadFailed(_name);
        }

        // Return the loaded data.
        return result;
    }

    /// @notice Attempts to load a proxy from a source function where the proxy should be found. If
    ///         the proxy isn't found at the source, or the call to the source fails, we build a
    ///         new proxy instead. Calls to source contracts MUST NOT fail under any circumstances
    ///         other than the function not existing (which can happen in an upgrade scenario).
    /// @param _source The source contract to load the proxy from.
    /// @param _selector The selector of the function to call on the source contract.
    /// @param _args The basic arguments for the proxy deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @param _instructions The extra upgrade instructions for the proxy deployment.
    /// @return The address of the loaded or built proxy.
    function loadOrDeployProxy(
        address _source,
        bytes4 _selector,
        ProxyDeployArgs memory _args,
        string memory _contractName,
        ExtraInstruction[] memory _instructions
    )
        external
        returns (address payable)
    {
        // Loads are allowed to fail ONLY if the user explicitly permitted it (or if this is a
        // deployment and the "ALL" permission is set).
        bool loadCanFail = hasInstruction(_instructions, Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, bytes(_contractName))
            || hasInstruction(
                _instructions, Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, Constants.PERMIT_ALL_CONTRACTS_INSTRUCTION
            );

        // Try to load the proxy from the source.
        (bool success, bytes memory result) = address(_source).staticcall(abi.encodePacked(_selector));

        // If the load succeeded, returned valid data, and the result is not a zero address, return the result.
        if (success && result.length >= 32 && abi.decode(result, (address)) != address(0)) {
            return payable(abi.decode(result, (address)));
        } else if (!loadCanFail) {
            // Load not permitted to fail but did, revert.
            revert OPContractsManagerUtils_ProxyMustLoad(_contractName);
        }

        // We've failed to load, but we allowed that failure.
        // Deploy the right proxy depending on the contract name.
        address ret;
        if (LibString.eq(_contractName, "L1StandardBridge")) {
            // L1StandardBridge is a special case ChugSplashProxy (legacy).
            ret = Blueprint.deployFrom(
                blueprints().l1ChugSplashProxy,
                computeSalt(_args.l2ChainId, _args.saltMixer, "L1StandardBridge"),
                abi.encode(_args.proxyAdmin)
            );

            // ChugSplashProxy requires setting the proxy type on the ProxyAdmin.
            _args.proxyAdmin.setProxyType(ret, IProxyAdmin.ProxyType.CHUGSPLASH);
        } else if (LibString.eq(_contractName, "L1CrossDomainMessenger")) {
            // L1CrossDomainMessenger is a special case ResolvedDelegateProxy (legacy).
            string memory l1XdmName = "OVM_L1CrossDomainMessenger";
            ret = Blueprint.deployFrom(
                blueprints().resolvedDelegateProxy,
                computeSalt(_args.l2ChainId, _args.saltMixer, "L1CrossDomainMessenger"),
                abi.encode(_args.addressManager, l1XdmName)
            );

            // ResolvedDelegateProxy requires setting the proxy type on the ProxyAdmin.
            _args.proxyAdmin.setProxyType(ret, IProxyAdmin.ProxyType.RESOLVED);
            _args.proxyAdmin.setImplementationName(ret, l1XdmName);
        } else {
            // Otherwise this is a normal proxy.
            ret = Blueprint.deployFrom(
                blueprints().proxy,
                computeSalt(_args.l2ChainId, _args.saltMixer, _contractName),
                abi.encode(_args.proxyAdmin)
            );
        }

        // Emit the proxy creation event.
        emit ProxyCreation(_contractName, ret);

        // Return the final deployment result.
        return payable(ret);
    }

    /// @notice Upgrades a contract by resetting the initialized slot and calling the initializer.
    /// @param _proxyAdmin The proxy admin of the contract.
    /// @param _target The target of the contract.
    /// @param _implementation The implementation of the contract.
    /// @param _data The data to call the initializer with.
    /// @param _slot The slot where the initialized value is located.
    /// @param _offset The offset of the initializer value in the slot.
    function upgrade(
        IProxyAdmin _proxyAdmin,
        address _target,
        address _implementation,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        external
    {
        // Check to make sure that we're not downgrading. Downgrades aren't inherently dangerous
        // but we also don't test for them so we don't really know if a specific downgrade will be
        // dangerous or not. It's easier to just revert instead.
        // NOTE: We DO allow upgrades to the same version, which makes it possible to use this
        //       function to both upgrade and then later perform management actions like changing
        //       the prestate for the fault dispute games.
        if (
            _proxyAdmin.getProxyImplementation(payable(_target)) != address(0)
                && SemverComp.gt(ISemver(_target).version(), ISemver(_implementation).version())
        ) {
            revert OPContractsManagerUtils_DowngradeNotAllowed(address(_target));
        }

        // Block deployments with extra version tags (prerelease or build metadata) when dev
        // features are not enabled. Only clean X.Y.Z versions are allowed unless dev features are
        // explicitly enabled (which is already blocked on mainnet by the container constructor).
        if (
            contractsContainer.devFeatureBitmap() == bytes32(0)
                && SemverComp.hasExtraTag(ISemver(_implementation).version())
        ) {
            revert OPContractsManagerUtils_ExtraTagInProd(_implementation);
        }

        // Upgrade to StorageSetter.
        _proxyAdmin.upgrade(payable(_target), address(implementations().storageSetterImpl));

        // Otherwise, we need to reset the initialized slot and call the initializer.
        // Reset the initialized slot by zeroing the single byte at `_offset` (from the right).
        bytes32 current = IStorageSetter(_target).getBytes32(_slot);
        uint256 mask = ~(uint256(0xff) << (uint256(_offset) * 8));
        IStorageSetter(_target).setBytes32(_slot, bytes32(uint256(current) & mask));

        // Upgrade to the implementation and call the initializer.
        _proxyAdmin.upgradeAndCall(payable(address(_target)), _implementation, _data);
    }

    /// @notice Returns the implementations for the contracts.
    /// @return The implementations for the contracts.
    function implementations() public view returns (IOPContractsManagerContainer.Implementations memory) {
        return contractsContainer.implementations();
    }

    /// @notice Returns the blueprints for the contracts.
    /// @return The blueprints for the contracts.
    function blueprints() public view returns (IOPContractsManagerContainer.Blueprints memory) {
        return contractsContainer.blueprints();
    }

    /// @notice Helper for retrieving the dispute game implementation for a given game type.
    /// @param _gameType The game type to retrieve the implementation for.
    /// @return The dispute game implementation.
    function getGameImpl(GameType _gameType) public view returns (IDisputeGame) {
        IOPContractsManagerContainer.Implementations memory impls = implementations();
        if (_gameType.raw() == GameTypes.CANNON.raw()) {
            return IDisputeGame(impls.faultDisputeGameImpl);
        } else if (_gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
            return IDisputeGame(impls.permissionedDisputeGameImpl);
        } else if (_gameType.raw() == GameTypes.CANNON_KONA.raw()) {
            return IDisputeGame(impls.faultDisputeGameImpl);
        } else if (_gameType.raw() == GameTypes.SUPER_CANNON.raw()) {
            return IDisputeGame(impls.superFaultDisputeGameImpl);
        } else if (_gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()) {
            return IDisputeGame(impls.superPermissionedDisputeGameImpl);
        } else if (_gameType.raw() == GameTypes.SUPER_CANNON_KONA.raw()) {
            return IDisputeGame(impls.superFaultDisputeGameImpl);
        } else {
            revert IOPContractsManagerUtils.OPContractsManagerUtils_UnsupportedGameType();
        }
    }

    /// @notice Helper for creating game constructor arguments.
    /// @param _l2ChainId The L2 chain ID.
    /// @param _anchorStateRegistry The AnchorStateRegistry to use for dispute games.
    /// @param _delayedWETH The DelayedWETH to use for dispute games.
    /// @param _gcfg Configuration for the dispute game to create.
    /// @return The game constructor arguments.
    function makeGameArgs(
        uint256 _l2ChainId,
        IAnchorStateRegistry _anchorStateRegistry,
        IDelayedWETH _delayedWETH,
        IOPContractsManagerUtils.DisputeGameConfig memory _gcfg
    )
        public
        view
        returns (bytes memory)
    {
        IOPContractsManagerContainer.Implementations memory impls = implementations();
        if (
            _gcfg.gameType.raw() == GameTypes.CANNON.raw() || _gcfg.gameType.raw() == GameTypes.CANNON_KONA.raw()
                || _gcfg.gameType.raw() == GameTypes.SUPER_CANNON.raw()
                || _gcfg.gameType.raw() == GameTypes.SUPER_CANNON_KONA.raw()
        ) {
            IOPContractsManagerUtils.FaultDisputeGameConfig memory parsedInputArgs =
                abi.decode(_gcfg.gameArgs, (IOPContractsManagerUtils.FaultDisputeGameConfig));
            return abi.encodePacked(
                parsedInputArgs.absolutePrestate,
                impls.mipsImpl,
                address(_anchorStateRegistry),
                address(_delayedWETH),
                _l2ChainId
            );
        } else if (
            _gcfg.gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
                || _gcfg.gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()
        ) {
            IOPContractsManagerUtils.PermissionedDisputeGameConfig memory parsedInputArgs =
                abi.decode(_gcfg.gameArgs, (IOPContractsManagerUtils.PermissionedDisputeGameConfig));
            return abi.encodePacked(
                parsedInputArgs.absolutePrestate,
                impls.mipsImpl,
                address(_anchorStateRegistry),
                address(_delayedWETH),
                _l2ChainId,
                parsedInputArgs.proposer,
                parsedInputArgs.challenger
            );
        } else {
            revert IOPContractsManagerUtils.OPContractsManagerUtils_UnsupportedGameType();
        }
    }
}
