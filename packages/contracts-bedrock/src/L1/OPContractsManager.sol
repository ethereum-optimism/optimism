// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Blueprint } from "src/libraries/Blueprint.sol";
import { Constants } from "src/libraries/Constants.sol";

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { IResourceMetering } from "src/L1/interfaces/IResourceMetering.sol";
import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";

import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";

import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";

import { DelayedWETH } from "src/dispute/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { Claim, Duration, GameType, GameTypes } from "src/dispute/lib/Types.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

/// @custom:proxied true
contract OPContractsManager is ISemver, Initializable {
    // -------- Structs --------

    /// @notice Represents the roles that can be set when deploying a standard OP Stack chain.
    struct Roles {
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
    }

    /// @notice The full set of inputs to deploy a new OP Stack chain.
    struct DeployInput {
        Roles roles;
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint256 l2ChainId;
        // The correct type is AnchorStateRegistry.StartingAnchorRoot[] memory,
        // but OP Deployer does not yet support structs.
        bytes startingAnchorRoots;
    }

    /// @notice The full set of outputs from deploying a new OP Stack chain.
    struct DeployOutput {
        ProxyAdmin opChainProxyAdmin;
        AddressManager addressManager;
        L1ERC721Bridge l1ERC721BridgeProxy;
        SystemConfig systemConfigProxy;
        OptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        L1StandardBridge l1StandardBridgeProxy;
        L1CrossDomainMessenger l1CrossDomainMessengerProxy;
        // Fault proof contracts below.
        OptimismPortal2 optimismPortalProxy;
        DisputeGameFactory disputeGameFactoryProxy;
        AnchorStateRegistry anchorStateRegistryProxy;
        AnchorStateRegistry anchorStateRegistryImpl;
        FaultDisputeGame faultDisputeGame;
        PermissionedDisputeGame permissionedDisputeGame;
        DelayedWETH delayedWETHPermissionedGameProxy;
        DelayedWETH delayedWETHPermissionlessGameProxy;
    }

    /// @notice The logic address and initializer selector for an implementation contract.
    struct Implementation {
        address logic; // Address containing the deployed logic contract.
        bytes4 initializer; // Function selector for the initializer.
    }

    /// @notice Used to set the implementation for a contract by mapping a contract
    /// name to the implementation data.
    struct ImplementationSetter {
        string name; // Contract name.
        Implementation info; // Implementation to set.
    }

    /// @notice Addresses of ERC-5202 Blueprint contracts. There are used for deploying full size
    /// contracts, to reduce the code size of this factory contract. If it deployed full contracts
    /// using the `new Proxy()` syntax, the code size would get large fast, since this contract would
    /// contain the bytecode of every contract it deploys. Therefore we instead use Blueprints to
    /// reduce the code size of this contract.
    struct Blueprints {
        address addressManager;
        address proxy;
        address proxyAdmin;
        address l1ChugSplashProxy;
        address resolvedDelegateProxy;
        address anchorStateRegistry;
        address permissionedDisputeGame1;
        address permissionedDisputeGame2;
    }

    /// @notice Inputs required when initializing the OPContractsManager. To avoid 'StackTooDeep' errors,
    /// all necessary inputs (excluding immutables) for initialization are bundled together in this struct.
    struct InitializerInputs {
        Blueprints blueprints;
        ImplementationSetter[] setters;
        string release;
        bool isLatest;
    }

    // -------- Constants and Variables --------

    /// @custom:semver 1.0.0-beta.7
    string public constant version = "1.0.0-beta.7";

    /// @notice Represents the interface version so consumers know how to decode the DeployOutput struct
    /// that's emitted in the `Deployed` event. Whenever that struct changes, a new version should be used.
    uint256 public constant OUTPUT_VERSION = 0;

    /// @notice Address of the SuperchainConfig contract shared by all chains.
    SuperchainConfig public immutable superchainConfig;

    /// @notice Address of the ProtocolVersions contract shared by all chains.
    ProtocolVersions public immutable protocolVersions;

    /// @notice The latest release of the OP Contracts Manager, as a string of the format `op-contracts/vX.Y.Z`.
    string public latestRelease;

    /// @notice Maps a release version to a contract name to it's implementation data.
    mapping(string => mapping(string => Implementation)) public implementations;

    /// @notice Maps an L2 Chain ID to the SystemConfig for that chain.
    mapping(uint256 => SystemConfig) public systemConfigs;

    /// @notice Addresses of the Blueprint contracts.
    /// This is internal because if public the autogenerated getter method would return a tuple of
    /// addresses, but we want it to return a struct. This is also set via `initialize` because
    /// we can't make this an immutable variable as it is a non-value type.
    Blueprints internal blueprint;

    /// @notice Storage gap for future modifications, so we can expand the number of blueprints
    /// without affecting other storage variables.
    uint256[50] private __gap;

    // -------- Events --------

    /// @notice Emitted when a new OP Stack chain is deployed.
    /// @param outputVersion Version that indicates how to decode the `deployOutput` argument.
    /// @param l2ChainId Chain ID of the new chain.
    /// @param deployer Address that deployed the chain.
    /// @param deployOutput ABI-encoded output of the deployment.
    event Deployed(
        uint256 indexed outputVersion, uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput
    );

    // -------- Errors --------

    /// @notice Thrown when an address is the zero address.
    error AddressNotFound(address who);

    /// @notice Throw when a contract address has no code.
    error AddressHasNoCode(address who);

    /// @notice Thrown when a release version is already set.
    error AlreadyReleased();

    /// @notice Thrown when an invalid `l2ChainId` is provided to `deploy`.
    error InvalidChainId();

    /// @notice Thrown when a role's address is not valid.
    error InvalidRoleAddress(string role);

    /// @notice Thrown when the latest release is not set upon initialization.
    error LatestReleaseNotSet();

    // -------- Methods --------

    /// @notice OPCM is proxied. Therefore the `initialize` function replaces most constructor logic for this contract.

    constructor(SuperchainConfig _superchainConfig, ProtocolVersions _protocolVersions) {
        assertValidContractAddress(address(_superchainConfig));
        assertValidContractAddress(address(_protocolVersions));
        superchainConfig = _superchainConfig;
        protocolVersions = _protocolVersions;
        _disableInitializers();
    }

    function initialize(InitializerInputs memory _initializerInputs) public initializer {
        if (_initializerInputs.isLatest) latestRelease = _initializerInputs.release;
        if (keccak256(bytes(latestRelease)) == keccak256("")) revert LatestReleaseNotSet();

        for (uint256 i = 0; i < _initializerInputs.setters.length; i++) {
            ImplementationSetter memory setter = _initializerInputs.setters[i];
            Implementation storage impl = implementations[_initializerInputs.release][setter.name];
            if (impl.logic != address(0)) revert AlreadyReleased();

            impl.initializer = setter.info.initializer;
            impl.logic = setter.info.logic;
        }

        blueprint = _initializerInputs.blueprints;
    }

    function deploy(DeployInput calldata _input) external returns (DeployOutput memory) {
        assertValidInputs(_input);

        // TODO Determine how we want to choose salt, e.g. are we concerned about chain ID squatting
        // since this approach means a chain ID can only be used once.
        uint256 l2ChainId = _input.l2ChainId;
        bytes32 salt = bytes32(_input.l2ChainId);
        DeployOutput memory output;

        // -------- Deploy Chain Singletons --------

        // The ProxyAdmin is the owner of all proxies for the chain. We temporarily set the owner to
        // this contract, and then transfer ownership to the specified owner at the end of deployment.
        // The AddressManager is used to store the implementation for the L1CrossDomainMessenger
        // due to it's usage of the legacy ResolvedDelegateProxy.
        output.addressManager = AddressManager(Blueprint.deployFrom(blueprint.addressManager, salt));
        output.opChainProxyAdmin =
            ProxyAdmin(Blueprint.deployFrom(blueprint.proxyAdmin, salt, abi.encode(address(this))));
        output.opChainProxyAdmin.setAddressManager(output.addressManager);

        // -------- Deploy Proxy Contracts --------

        // Deploy ERC-1967 proxied contracts.
        output.l1ERC721BridgeProxy = L1ERC721Bridge(deployProxy(l2ChainId, output.opChainProxyAdmin, "L1ERC721Bridge"));
        output.optimismPortalProxy =
            OptimismPortal2(payable(deployProxy(l2ChainId, output.opChainProxyAdmin, "OptimismPortal")));
        output.systemConfigProxy = SystemConfig(deployProxy(l2ChainId, output.opChainProxyAdmin, "SystemConfig"));
        output.optimismMintableERC20FactoryProxy = OptimismMintableERC20Factory(
            deployProxy(l2ChainId, output.opChainProxyAdmin, "OptimismMintableERC20Factory")
        );
        output.disputeGameFactoryProxy =
            DisputeGameFactory(deployProxy(l2ChainId, output.opChainProxyAdmin, "DisputeGameFactory"));
        output.anchorStateRegistryProxy =
            AnchorStateRegistry(deployProxy(l2ChainId, output.opChainProxyAdmin, "AnchorStateRegistry"));

        // Deploy legacy proxied contracts.
        output.l1StandardBridgeProxy = L1StandardBridge(
            payable(Blueprint.deployFrom(blueprint.l1ChugSplashProxy, salt, abi.encode(output.opChainProxyAdmin)))
        );
        output.opChainProxyAdmin.setProxyType(address(output.l1StandardBridgeProxy), ProxyAdmin.ProxyType.CHUGSPLASH);

        string memory contractName = "OVM_L1CrossDomainMessenger";
        output.l1CrossDomainMessengerProxy = L1CrossDomainMessenger(
            Blueprint.deployFrom(blueprint.resolvedDelegateProxy, salt, abi.encode(output.addressManager, contractName))
        );
        output.opChainProxyAdmin.setProxyType(
            address(output.l1CrossDomainMessengerProxy), ProxyAdmin.ProxyType.RESOLVED
        );
        output.opChainProxyAdmin.setImplementationName(address(output.l1CrossDomainMessengerProxy), contractName);

        // Now that all proxies are deployed, we can transfer ownership of the AddressManager to the ProxyAdmin.
        output.addressManager.transferOwnership(address(output.opChainProxyAdmin));

        // The AnchorStateRegistry Implementation is not MCP Ready, and therefore requires an implementation per chain.
        // It must be deployed after the DisputeGameFactoryProxy so that it can be provided as a constructor argument.
        output.anchorStateRegistryImpl = AnchorStateRegistry(
            Blueprint.deployFrom(blueprint.anchorStateRegistry, salt, abi.encode(output.disputeGameFactoryProxy))
        );

        // We have two delayed WETH contracts per chain, one for each of the permissioned and permissionless games.
        output.delayedWETHPermissionlessGameProxy =
            DelayedWETH(payable(deployProxy(l2ChainId, output.opChainProxyAdmin, "DelayedWETHPermissionlessGame")));
        output.delayedWETHPermissionedGameProxy =
            DelayedWETH(payable(deployProxy(l2ChainId, output.opChainProxyAdmin, "DelayedWETHPermissionedGame")));

        // While not a proxy, we deploy the PermissionedDisputeGame here as well because it's bespoke per chain.
        output.permissionedDisputeGame = PermissionedDisputeGame(
            Blueprint.deployFrom(
                blueprint.permissionedDisputeGame1,
                blueprint.permissionedDisputeGame2,
                salt,
                encodePermissionedDisputeGameConstructor(_input, output)
            )
        );

        // -------- Set and Initialize Proxy Implementations --------
        Implementation memory impl;
        bytes memory data;

        impl = getLatestImplementation("L1ERC721Bridge");
        data = encodeL1ERC721BridgeInitializer(impl.initializer, output);
        upgradeAndCall(output.opChainProxyAdmin, address(output.l1ERC721BridgeProxy), impl.logic, data);

        impl = getLatestImplementation("OptimismPortal");
        data = encodeOptimismPortalInitializer(impl.initializer, output);
        upgradeAndCall(output.opChainProxyAdmin, address(output.optimismPortalProxy), impl.logic, data);

        impl = getLatestImplementation("SystemConfig");
        data = encodeSystemConfigInitializer(impl.initializer, _input, output);
        upgradeAndCall(output.opChainProxyAdmin, address(output.systemConfigProxy), impl.logic, data);

        impl = getLatestImplementation("OptimismMintableERC20Factory");
        data = encodeOptimismMintableERC20FactoryInitializer(impl.initializer, output);
        upgradeAndCall(output.opChainProxyAdmin, address(output.optimismMintableERC20FactoryProxy), impl.logic, data);

        impl = getLatestImplementation("L1CrossDomainMessenger");
        data = encodeL1CrossDomainMessengerInitializer(impl.initializer, output);
        upgradeAndCall(output.opChainProxyAdmin, address(output.l1CrossDomainMessengerProxy), impl.logic, data);

        impl = getLatestImplementation("L1StandardBridge");
        data = encodeL1StandardBridgeInitializer(impl.initializer, output);
        upgradeAndCall(output.opChainProxyAdmin, address(output.l1StandardBridgeProxy), impl.logic, data);

        impl = getLatestImplementation("DelayedWETH");
        data = encodeDelayedWETHInitializer(impl.initializer, _input);
        upgradeAndCall(output.opChainProxyAdmin, address(output.delayedWETHPermissionedGameProxy), impl.logic, data);
        upgradeAndCall(output.opChainProxyAdmin, address(output.delayedWETHPermissionlessGameProxy), impl.logic, data);

        // We set the initial owner to this contract, set game implementations, then transfer ownership.
        impl = getLatestImplementation("DisputeGameFactory");
        data = encodeDisputeGameFactoryInitializer(impl.initializer, _input);
        upgradeAndCall(output.opChainProxyAdmin, address(output.disputeGameFactoryProxy), impl.logic, data);
        output.disputeGameFactoryProxy.setImplementation(
            GameTypes.PERMISSIONED_CANNON, IDisputeGame(address(output.permissionedDisputeGame))
        );
        output.disputeGameFactoryProxy.setInitBond(GameTypes.PERMISSIONED_CANNON, 0.08 ether);
        output.disputeGameFactoryProxy.transferOwnership(address(output.opChainProxyAdmin));

        impl.logic = address(output.anchorStateRegistryImpl);
        impl.initializer = AnchorStateRegistry.initialize.selector;
        data = encodeAnchorStateRegistryInitializer(impl.initializer, _input);
        upgradeAndCall(output.opChainProxyAdmin, address(output.anchorStateRegistryProxy), impl.logic, data);

        // -------- Finalize Deployment --------
        // Transfer ownership of the ProxyAdmin from this contract to the specified owner.
        output.opChainProxyAdmin.transferOwnership(_input.roles.opChainProxyAdminOwner);

        emit Deployed(OUTPUT_VERSION, l2ChainId, msg.sender, abi.encode(output));
        return output;
    }

    // -------- Utilities --------

    /// @notice Verifies that all inputs are valid and reverts if any are invalid.
    /// Typically the proxy admin owner is expected to have code, but this is not enforced here.
    function assertValidInputs(DeployInput calldata _input) internal view {
        if (_input.l2ChainId == 0 || _input.l2ChainId == block.chainid) revert InvalidChainId();

        if (_input.roles.opChainProxyAdminOwner == address(0)) revert InvalidRoleAddress("opChainProxyAdminOwner");
        if (_input.roles.systemConfigOwner == address(0)) revert InvalidRoleAddress("systemConfigOwner");
        if (_input.roles.batcher == address(0)) revert InvalidRoleAddress("batcher");
        if (_input.roles.unsafeBlockSigner == address(0)) revert InvalidRoleAddress("unsafeBlockSigner");
        if (_input.roles.proposer == address(0)) revert InvalidRoleAddress("proposer");
        if (_input.roles.challenger == address(0)) revert InvalidRoleAddress("challenger");
    }

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    /// configuration's convention. This convention is `versionByte || keccak256(bytes32(chainId))[:19]`,
    /// where || denotes concatenation`, versionByte is 0x00, and chainId is a uint256.
    /// https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) public pure returns (address) {
        bytes1 versionByte = 0x00;
        bytes32 hashedChainId = keccak256(bytes.concat(bytes32(_l2ChainId)));
        bytes19 first19Bytes = bytes19(hashedChainId);
        return address(uint160(bytes20(bytes.concat(versionByte, first19Bytes))));
    }

    /// @notice Deterministically deploys a new proxy contract owned by the provided ProxyAdmin.
    /// The salt is computed as a function of the L2 chain ID and the contract name. This is required
    /// because we deploy many identical proxies, so they each require a unique salt for determinism.
    function deployProxy(
        uint256 _l2ChainId,
        ProxyAdmin _proxyAdmin,
        string memory _contractName
    )
        internal
        returns (address)
    {
        bytes32 salt = keccak256(abi.encode(_l2ChainId, _contractName));
        return Blueprint.deployFrom(blueprint.proxy, salt, abi.encode(_proxyAdmin));
    }

    /// @notice Returns the implementation data for a contract name. Makes a copy of the internal
    //  Implementation struct in storage to prevent accidental mutation of the internal data.
    function getLatestImplementation(string memory _name) internal view returns (Implementation memory) {
        Implementation storage impl = implementations[latestRelease][_name];
        return Implementation({ logic: impl.logic, initializer: impl.initializer });
    }

    // -------- Initializer Encoding --------

    /// @notice Helper method for encoding the L1ERC721Bridge initializer data.
    function encodeL1ERC721BridgeInitializer(
        bytes4 _selector,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        return abi.encodeWithSelector(_selector, _output.l1CrossDomainMessengerProxy, superchainConfig);
    }

    /// @notice Helper method for encoding the OptimismPortal initializer data.
    function encodeOptimismPortalInitializer(
        bytes4 _selector,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        _output;
        // TODO make GameTypes.CANNON an input once FPs are supported
        return abi.encodeWithSelector(
            _selector,
            _output.disputeGameFactoryProxy,
            _output.systemConfigProxy,
            superchainConfig,
            GameTypes.PERMISSIONED_CANNON
        );
    }

    /// @notice Helper method for encoding the SystemConfig initializer data.
    function encodeSystemConfigInitializer(
        bytes4 selector,
        DeployInput memory _input,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        (ResourceMetering.ResourceConfig memory referenceResourceConfig, SystemConfig.Addresses memory opChainAddrs) =
            defaultSystemConfigParams(selector, _input, _output);

        return abi.encodeWithSelector(
            selector,
            _input.roles.systemConfigOwner,
            _input.basefeeScalar,
            _input.blobBasefeeScalar,
            bytes32(uint256(uint160(_input.roles.batcher))), // batcherHash
            30_000_000, // gasLimit, TODO should this be an input?
            _input.roles.unsafeBlockSigner,
            referenceResourceConfig,
            chainIdToBatchInboxAddress(_input.l2ChainId),
            opChainAddrs
        );
    }

    /// @notice Helper method for encoding the OptimismMintableERC20Factory initializer data.
    function encodeOptimismMintableERC20FactoryInitializer(
        bytes4 _selector,
        DeployOutput memory _output
    )
        internal
        pure
        virtual
        returns (bytes memory)
    {
        return abi.encodeWithSelector(_selector, _output.l1StandardBridgeProxy);
    }

    /// @notice Helper method for encoding the L1CrossDomainMessenger initializer data.
    function encodeL1CrossDomainMessengerInitializer(
        bytes4 _selector,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        return
            abi.encodeWithSelector(_selector, superchainConfig, _output.optimismPortalProxy, _output.systemConfigProxy);
    }

    /// @notice Helper method for encoding the L1StandardBridge initializer data.
    function encodeL1StandardBridgeInitializer(
        bytes4 _selector,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        return abi.encodeWithSelector(
            _selector, _output.l1CrossDomainMessengerProxy, superchainConfig, _output.systemConfigProxy
        );
    }

    function encodeDisputeGameFactoryInitializer(
        bytes4 _selector,
        DeployInput memory
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        // This contract must be the initial owner so we can set game implementations, then
        // ownership is transferred after.
        return abi.encodeWithSelector(_selector, address(this));
    }

    function encodeAnchorStateRegistryInitializer(
        bytes4 _selector,
        DeployInput memory _input
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        // this line fails in the op-deployer tests because it is not passing in any data
        AnchorStateRegistry.StartingAnchorRoot[] memory startingAnchorRoots =
            abi.decode(_input.startingAnchorRoots, (AnchorStateRegistry.StartingAnchorRoot[]));
        return abi.encodeWithSelector(_selector, startingAnchorRoots, superchainConfig);
    }

    function encodeDelayedWETHInitializer(
        bytes4 _selector,
        DeployInput memory _input
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        return abi.encodeWithSelector(_selector, _input.roles.opChainProxyAdminOwner, superchainConfig);
    }

    function encodePermissionedDisputeGameConstructor(
        DeployInput memory _input,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (bytes memory)
    {
        return abi.encode(
            GameType.wrap(1), // Permissioned Cannon
            Claim.wrap(bytes32(hex"dead")), // absolutePrestate
            73, // maxGameDepth
            30, // splitDepth
            Duration.wrap(3 hours), // clockExtension
            Duration.wrap(3.5 days), // maxClockDuration
            IBigStepper(getLatestImplementation("MIPS").logic),
            IDelayedWETH(payable(address(_output.delayedWETHPermissionedGameProxy))),
            IAnchorStateRegistry(address(_output.anchorStateRegistryProxy)),
            _input.l2ChainId,
            _input.roles.proposer,
            _input.roles.challenger
        );
    }

    /// @notice Returns default, standard config arguments for the SystemConfig initializer.
    /// This is used by subclasses to reduce code duplication.
    function defaultSystemConfigParams(
        bytes4, /* selector */
        DeployInput memory, /* _input */
        DeployOutput memory _output
    )
        internal
        view
        virtual
        returns (ResourceMetering.ResourceConfig memory resourceConfig_, SystemConfig.Addresses memory opChainAddrs_)
    {
        // We use assembly to easily convert from IResourceMetering.ResourceConfig to ResourceMetering.ResourceConfig.
        // This is required because we have not yet fully migrated the codebase to be interface-based.
        IResourceMetering.ResourceConfig memory resourceConfig = Constants.DEFAULT_RESOURCE_CONFIG();
        assembly ("memory-safe") {
            resourceConfig_ := resourceConfig
        }

        opChainAddrs_ = SystemConfig.Addresses({
            l1CrossDomainMessenger: address(_output.l1CrossDomainMessengerProxy),
            l1ERC721Bridge: address(_output.l1ERC721BridgeProxy),
            l1StandardBridge: address(_output.l1StandardBridgeProxy),
            disputeGameFactory: address(_output.disputeGameFactoryProxy),
            optimismPortal: address(_output.optimismPortalProxy),
            optimismMintableERC20Factory: address(_output.optimismMintableERC20FactoryProxy),
            gasPayingToken: Constants.ETHER
        });

        assertValidContractAddress(opChainAddrs_.l1CrossDomainMessenger);
        assertValidContractAddress(opChainAddrs_.l1ERC721Bridge);
        assertValidContractAddress(opChainAddrs_.l1StandardBridge);
        assertValidContractAddress(opChainAddrs_.disputeGameFactory);
        assertValidContractAddress(opChainAddrs_.optimismPortal);
        assertValidContractAddress(opChainAddrs_.optimismMintableERC20Factory);
    }

    /// @notice Makes an external call to the target to initialize the proxy with the specified data.
    /// First performs safety checks to ensure the target, implementation, and proxy admin are valid.
    function upgradeAndCall(
        ProxyAdmin _proxyAdmin,
        address _target,
        address _implementation,
        bytes memory _data
    )
        internal
    {
        assertValidContractAddress(address(_proxyAdmin));
        assertValidContractAddress(_target);
        assertValidContractAddress(_implementation);

        _proxyAdmin.upgradeAndCall(payable(address(_target)), _implementation, _data);
    }

    function assertValidContractAddress(address _who) internal view {
        if (_who == address(0)) revert AddressNotFound(_who);
        if (_who.code.length == 0) revert AddressHasNoCode(_who);
    }

    /// @notice Returns the blueprint contract addresses.
    function blueprints() public view returns (Blueprints memory) {
        return blueprint;
    }
}
