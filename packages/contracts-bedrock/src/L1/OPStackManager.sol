// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { AddressManager } from "src/legacy/AddressManager.sol";
import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";

/// @notice The OPStackManager deploys a new OP Chain that complies with a valid "standard
/// configuration". The Optimism Specs define configurable properties for a chain. If configuration
/// values meet certain requirements this chain meets the "standard configuration" Superchain Level.
///
/// Some required values for the standard configuration can be discovered onchain by reading those
/// values from a chain known to meet the standard configuration requirements. This chain is known
/// as the "reference chain" in this contract, and is why many variables are prefixed with
/// "reference". The reference chain will be OP Mainnet, OP Sepolia, etc.
///
/// OP Stack chains deployed by this contract have the following properties:
///
///   1. Chain IDs are no larger than `type(uint64).max`. This allows a standard batch inbox address
///      format to be computed and enforced at deployment.
///   2. All contracts are deployed using CREATE2 with a salt that is a function of L2 chain ID. As
///      a result, all contract addresses for a chain deployed by this contract can be
///      deterministically (and even counterfactually) computed by any party.
///        - TODO Consider using CREATE3 so addresses are independent of bytecode, to be robust
///          against future code changes. This is mainly important for proxies and contracts that
///          are not deployed behind a proxy, such as the AddressManager.
///        - TODO Add getter methods to help compute all contract addresses for a chain.
///   3. When practical, standard chain requirements are automatically set or enforced, such as
///      having a block time of 2 seconds.
///   4. They use the `referenceSuperchainConfig` and `referenceProtocolVersions` contracts, since
///      these are intended to be singletons. All chains pointing to the `referenceSuperchainConfig`
///      can have withdrawals paused simultaneously, which is a security feature that assists with
///      incident response.
contract OPStackManager is OwnableUpgradeable {
    /// @notice The reference OP SystemConfig used for reference values.
    SystemConfig public immutable referenceSystemConfig;

    /// @notice The reference OP AddressManager used for reference values.
    AddressManager public immutable referenceAddressManager;

    /// @notice The owner of the ProxyAdmin for the reference OP chain.
    address public immutable referenceProxyAdminOwner;

    /// @notice The reference OP SuperchainConfig used by all chains deployed by this contract.
    SuperchainConfig public immutable referenceSuperchainConfig;

    /// @notice The reference OP ProtocolVersions used by all chains deployed by this contract.
    ProtocolVersions public immutable referenceProtocolVersions;

    /// @notice Used to store the addresses of the deployed proxies for a chain,
    /// to mitigate stack too deep errors.
    struct Proxies {
        address l1ERC721Bridge;
        address l2OutputOracle;
        address optimismPortal;
        address systemConfig;
        address optimismMintableERC20Factory;
        address l1CrossDomainMessenger;
        address l1StandardBridge;
    }

    /// @notice Inputs required to initialize the SystemConfig contract for a new chain.
    struct SystemConfigInputs {
        address systemConfigOwner;
        uint256 overhead;
        uint256 scalar;
        bytes32 batcherHash;
        address unsafeBlockSigner;
    }

    /// @notice Inputs required to initialize the L2OutputOracle contract for a new chain.
    struct L2OutputOracleInputs {
        uint256 submissionInterval;
        address proposer;
        address challenger;
    }

    /// @notice The logic address and initializer selector for an implementation contract.
    struct Implementation {
        address logic; // Address containing the deployed logic contract.
        bytes4 initializer; // Function selector for the initializer.
    }

    /// @notice Used to set the implementation for a contract by mapping a contract
    /// name to the implementation data.
    struct ImplementationSetter {
        bytes32 name; // Contract name.
        Implementation info; // Implementation to set.
    }

    /// @notice Returns the latest approved release of the OP Stack contracts.
    /// Release strings follow semver and are named with the format `op-contracts/vX.Y.Z`.
    bytes32 public latestRelease;

    /// @notice Maps a release version to a contract name to it's implementation data.
    mapping(bytes32 => mapping(bytes32 => Implementation)) public implementations;

    /// @notice Maps an L2 Chain ID to the SystemConfig for that chain.
    /// Most information for a chain can be found from it's the SystemConfig, with ProtocolVersions
    /// being the exception. ProtocolVersions is a singleton for all chains deployed by this
    /// contract, so `referenceProtocolVersions` is used for all chains.
    /// Additionally, the ProxyAdmin can be discovered offchain, but cannot be discovered onchain
    /// due to implementation details of the ProxyAdmin contract.
    mapping(uint256 => SystemConfig) public systemConfigs;

    /// @notice Maps an L2 Chain ID to the release version for that chain.
    /// Release strings follow semver and are named with the format `op-contracts/vX.Y.Z`.
    mapping(uint256 => bytes32) public releases;

    constructor(
        uint64 referenceChainId,
        SystemConfig _referenceSystemConfig,
        ProtocolVersions _referenceProtocolVersions,
        AddressManager _referenceAddressManager,
        address _referenceProxyAdminOwner
    ) {
        _disableInitializers();

        referenceSystemConfig = _referenceSystemConfig;
        referenceProtocolVersions = _referenceProtocolVersions;

        address referencePortal = referenceSystemConfig.optimismPortal();
        referenceSuperchainConfig = OptimismPortal(payable(referencePortal)).superchainConfig();
        referenceAddressManager = _referenceAddressManager;
        referenceProxyAdminOwner = _referenceProxyAdminOwner;

        register(referenceChainId, latestRelease, referenceSystemConfig);
    }

    function initialize(address _owner) external initializer {
        __Ownable_init();
        transferOwnership(_owner);
    }

    /// @notice Called by the OP Stack Manager owner to release a set of implementation contracts.
    function release(bytes32 version, bool isLatest, ImplementationSetter[] calldata impls) public onlyOwner {
        for (uint256 i = 0; i < impls.length; i++) {
            ImplementationSetter calldata implSetter = impls[i];
            Implementation storage impl = implementations[version][implSetter.name];
            require(impl.logic == address(0), "OpStackManager: Implementation already exists");

            impl.initializer = implSetter.info.initializer;
            impl.logic = implSetter.info.logic;
        }

        if (isLatest) {
            latestRelease = version;
        }
    }

    /// @notice Used to make this contract aware of chains that were deployed without this factory.
    /// This method is permissionless, and anyone can register any chain ID.
    function register(uint64 l2ChainId, bytes32 release_, SystemConfig sc) public {
        requireValidL2ChainId(l2ChainId);
        // TODO Add other standard configuration requirement checks here.
        systemConfigs[l2ChainId] = sc;
        releases[l2ChainId] = release_;
    }

    /// @notice Similar to `register`, but only callable by the owner to bypass chain ID checks.
    /// This is required because the source of truth for L2 Chain IDs lives offchain, meaning it's
    /// possible for malicious actors to squat on chain IDs, therefore the owner can fix or delete
    /// invalid registrations.
    function registerOverride(uint256 l2ChainId, bytes32 release_, SystemConfig sc) public onlyOwner {
        // TODO Add standard configuration requirement checks here.
        systemConfigs[l2ChainId] = sc;
        releases[l2ChainId] = release_;
    }

    /// @notice Deploys a new chain with the given properties.
    /// This function signature is not expected to stay backwards compatible, as initializers and
    /// required inputs may change over time.
    ///
    /// WARNING: Due to using the L2 chain ID as salt for CREATE2 usage for proxy deployments, if a
    /// squatter deploys an illegitimate chain using a given chain ID, even this contract's owner
    /// cannot use this method to redeploy that chain. It's recommend to not have any allegiance to
    /// a specific chain ID so you can instead simply choose another. But if a certain chain ID is
    /// required, after squatting via this method the deployments options become:
    ///   1. Deploy the chain outside of this contract and register it separately. With this method,
    ///      that chain's contract addresses are no longer a deterministic function of the chain ID.
    ///      Additionally, care must be used to ensure the chain is properly deployed and configured
    ///      to meet the standard configuration requirements.
    ///   2. Use this method to deploy with an arbitrary chain ID, and request this contract's owner
    ///      to use the `registerOverride` method to re-register the chain with the correct chain ID.
    ///      This chain will meet the standard configuration requirements, but it's contract
    ///      addressees will be deterministic based on an arbitrary chain ID instead of the actual
    ///      chain ID.
    function deploy(
        uint64 l2ChainId,
        address proxyAdminOwner,
        SystemConfigInputs memory systemConfigInputs,
        L2OutputOracleInputs memory l2OutputOracleInputs
    )
        external
    {
        // -------- Requirements --------
        requireValidL2ChainId(l2ChainId);
        // TODO Add other standard configuration requirement checks here.

        // -------- Deploy AddressManager and ProxyAdmin --------
        bytes32 salt = bytes32(uint256(l2ChainId));

        // The ProxyAdmin is the owner of all proxies for the chain. We temporarily set the owner to
        // this contract, and then transfer ownership to the specified owner at the end of deployment.
        // The AddressManager is used to store the implementation for the L1CrossDomainMessenger
        // due to it's usage of the legacy ResolvedDelegateProxy.
        AddressManager addressManager = new AddressManager{ salt: salt }();
        ProxyAdmin proxyAdmin = new ProxyAdmin{ salt: salt }({ _owner: address(this) });
        proxyAdmin.setAddressManager(addressManager);

        // -------- Deploy Proxies --------
        Proxies memory proxies;

        // Deploy ERC-1967 proxied contracts.
        proxies.l1ERC721Bridge = _deployProxy(l2ChainId, proxyAdmin, "L1ERC721Bridge");
        proxies.l2OutputOracle = _deployProxy(l2ChainId, proxyAdmin, "L2OutputOracle");
        proxies.optimismPortal = _deployProxy(l2ChainId, proxyAdmin, "OptimismPortal");
        proxies.systemConfig = _deployProxy(l2ChainId, proxyAdmin, "SystemConfig");
        proxies.optimismMintableERC20Factory = _deployProxy(l2ChainId, proxyAdmin, "OptimismMintableERC20Factory");

        // Deploy legacy proxied contracts.
        proxies.l1StandardBridge = address(new L1ChugSplashProxy{ salt: salt }(address(proxyAdmin)));
        proxyAdmin.setProxyType(proxies.l1StandardBridge, ProxyAdmin.ProxyType.CHUGSPLASH);

        string memory contractName = "OVM_L1CrossDomainMessenger";
        proxies.l1CrossDomainMessenger = address(new ResolvedDelegateProxy{ salt: salt }(addressManager, contractName));
        proxyAdmin.setProxyType(proxies.l1CrossDomainMessenger, ProxyAdmin.ProxyType.RESOLVED);
        proxyAdmin.setImplementationName(proxies.l1CrossDomainMessenger, contractName);

        // Now that all proxies are deployed, we can transfer ownership of the AddressManager to
        // the ProxyAdmin.
        addressManager.transferOwnership(address(proxyAdmin));

        // -------- Set and Initialize Proxy Implementations --------
        Implementation storage impl;
        bytes memory data;

        impl = _getLatestImplementation("L1ERC721Bridge");
        data = abi.encodeWithSelector(impl.initializer, proxies.l1CrossDomainMessenger, referenceSuperchainConfig);
        proxyAdmin.upgradeAndCall(payable(proxies.l1ERC721Bridge), impl.logic, data);

        impl = _getLatestImplementation("L2OutputOracle");
        L2OutputOracle referenceL2OO = L2OutputOracle(referenceSystemConfig.l2OutputOracle());
        data = _encodeL2OOInitializer(impl.initializer, l2OutputOracleInputs, referenceL2OO);
        proxyAdmin.upgradeAndCall(payable(proxies.l2OutputOracle), impl.logic, data);

        impl = _getLatestImplementation("OptimismPortal");
        data = _encodeOptimismPortalInitializer(impl.initializer, proxies);
        proxyAdmin.upgradeAndCall(payable(proxies.optimismPortal), impl.logic, data);

        impl = _getLatestImplementation("SystemConfig");
        data = _encodeSystemConfigInitializer(impl.initializer, l2ChainId, systemConfigInputs, proxies);
        proxyAdmin.upgradeAndCall(payable(proxies.systemConfig), impl.logic, data);

        impl = _getLatestImplementation("OptimismMintableERC20Factory");
        data = abi.encodeWithSelector(impl.initializer, proxies.l1StandardBridge);
        proxyAdmin.upgradeAndCall(payable(proxies.optimismMintableERC20Factory), impl.logic, data);

        impl = _getLatestImplementation("L1CrossDomainMessenger");
        require(
            impl.logic == referenceAddressManager.getAddress("OVM_L1CrossDomainMessenger"),
            "OpStackManager: L1CrossDomainMessenger implementation mismatch"
        );
        data = abi.encodeWithSelector(impl.initializer, referenceSuperchainConfig, proxies.optimismPortal);
        proxyAdmin.upgradeAndCall(payable(proxies.l1CrossDomainMessenger), impl.logic, data);

        // -------- Finalize Deployment --------
        // Transfer ownership of the ProxyAdmin from this contract to the specified owner.
        proxyAdmin.transferOwnership(proxyAdminOwner);

        // Save off this deploy.
        register(l2ChainId, latestRelease, SystemConfig(proxies.systemConfig));

        // Correctness checks.
        // forgefmt: disable-start
        SystemConfig systemConfig = SystemConfig(proxies.systemConfig);
        require(systemConfig.owner() == systemConfigInputs.systemConfigOwner, "OpStackManager: SystemConfig owner mismatch");
        require(systemConfig.l1CrossDomainMessenger() == proxies.l1CrossDomainMessenger, "OpStackManager: L1CrossDomainMessenger mismatch");
        require(systemConfig.l1ERC721Bridge() == proxies.l1ERC721Bridge, "OpStackManager: L1ERC721Bridge mismatch");
        require(systemConfig.l1StandardBridge() == proxies.l1StandardBridge, "OpStackManager: L1StandardBridge mismatch");
        require(systemConfig.l2OutputOracle() == proxies.l2OutputOracle, "OpStackManager: L2OutputOracle mismatch");
        require(systemConfig.optimismPortal() == proxies.optimismPortal, "OpStackManager: OptimismPortal mismatch");
        require(systemConfig.optimismMintableERC20Factory() == proxies.optimismMintableERC20Factory, "OpStackManager: OptimismMintableERC20Factory mismatch");
        // forgefmt: disable-end
    }

    /// @notice Maps an L2 chain ID to an L1 batch inbox address of the form `0xFF000...000{chainId}`.
    function chainIdToBatchInboxAddress(uint256 l2ChainId) public pure returns (address) {
        return address(uint160(0xFF) << 152 | uint160(l2ChainId));
    }

    /// @notice Helper method for encoding the OptimismPortal initializer data.
    function _encodeOptimismPortalInitializer(
        bytes4 selector,
        Proxies memory proxies
    )
        internal
        view
        returns (bytes memory)
    {
        return abi.encodeWithSelector(selector, proxies.l2OutputOracle, proxies.systemConfig, referenceSystemConfig);
    }

    /// @notice Helper method for encoding the L2OutputOracle initializer data.
    function _encodeL2OOInitializer(
        bytes4 selector,
        L2OutputOracleInputs memory inputs,
        L2OutputOracle referenceL2OO
    )
        internal
        view
        returns (bytes memory)
    {
        return abi.encodeWithSelector(
            selector,
            inputs.submissionInterval,
            referenceL2OO.l2BlockTime(),
            block.number, // startingBlockNumber
            block.timestamp, // startingTimestamp
            inputs.proposer,
            inputs.challenger,
            referenceL2OO.finalizationPeriodSeconds()
        );
    }

    /// @notice Helper method for encoding the SystemConfig initializer data.
    function _encodeSystemConfigInitializer(
        bytes4 selector,
        uint64 l2ChainId,
        SystemConfigInputs memory inputs,
        Proxies memory proxies
    )
        internal
        pure
        returns (bytes memory)
    {
        ResourceMetering.ResourceConfig memory referenceResourceConfig = ResourceMetering.ResourceConfig({
            maxResourceLimit: 2e7,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            minimumBaseFee: 1e9,
            systemTxMaxGas: 1e6,
            maximumBaseFee: 340282366920938463463374607431768211455
        });

        SystemConfig.Addresses memory _addresses = SystemConfig.Addresses({
            l1CrossDomainMessenger: proxies.l1CrossDomainMessenger,
            l1ERC721Bridge: proxies.l1ERC721Bridge,
            l1StandardBridge: proxies.l1StandardBridge,
            l2OutputOracle: proxies.l2OutputOracle,
            optimismPortal: proxies.optimismPortal,
            optimismMintableERC20Factory: proxies.optimismMintableERC20Factory
        });

        return abi.encodeWithSelector(
            selector,
            inputs.systemConfigOwner,
            inputs.overhead,
            inputs.scalar,
            inputs.batcherHash,
            30_000_000, // gasLimit
            inputs.unsafeBlockSigner,
            referenceResourceConfig,
            chainIdToBatchInboxAddress(l2ChainId),
            _addresses
        );
    }

    /// @notice Deterministically deploys a new proxy contract. The salt is computed as a function
    /// of the L2 chain ID and the contract name. This is required because we deploy many identical
    /// proxies, so they each require a unique salt for determinism.
    function _deployProxy(uint64 l2ChainId, ProxyAdmin proxyAdmin, bytes32 contractName) internal returns (address) {
        bytes32 salt = keccak256(abi.encode(l2ChainId, contractName));
        return address(new Proxy{ salt: salt }(address(proxyAdmin)));
    }

    /// @notice Returns the implementation data for a contract name.
    function _getLatestImplementation(bytes32 name) internal view returns (Implementation storage) {
        return implementations[latestRelease][name];
    }

    /// @notice Reverts if the given L2 chain ID is invalid.
    function requireValidL2ChainId(uint64 l2ChainId) internal view {
        require(address(systemConfigs[l2ChainId]) == address(0), "OpStackManager: Already deployed");
        require(l2ChainId != 0 && l2ChainId != block.chainid, "OpStackManager: Invalid chain ID");
    }
}
