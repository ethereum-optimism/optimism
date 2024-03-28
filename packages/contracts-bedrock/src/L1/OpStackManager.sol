// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";

/// @dev WIP, lots of improvements needed.
contract OpStackManager is OwnableUpgradeable {
    SystemConfig public immutable standardSystemConfig;
    AddressManager public immutable standardAddressManager;
    SuperchainConfig public immutable standardSuperchainConfig;
    ProtocolVersions public immutable standardProtocolVersions;

    struct Proxies {
        address l1ERC721Bridge;
        address l2OutputOracle;
        address optimismPortal;
        address systemConfig;
        address optimismMintableERC20Factory;
        address l1CrossDomainMessenger;
        address l1StandardBridge;
    }

    struct Implementation {
        address logic; // Address containing the deployed logic contract.
        bytes4 initializer; // Function selector for the initializer.
    }

    struct ImplementationSetter {
        bytes32 name; // Contract name.
        Implementation info; // Implementation to set.
    }

    bytes32 public latestRelease; // Semver of the latest release, e.g. "v1.2.0" for the `op-contracts/v1.2.0` tag.
    mapping(bytes32 /* version */ => mapping(bytes32 /* name */ => Implementation)) public implementations;
    mapping(uint256 /* l2ChainId */ => SystemConfig) public systemConfigs; // Can find everything from SystemConfig...
    mapping(uint256 /* l2ChainId */ => ProtocolVersions) public protocolVersions; // ...except this. TODO can we add
        // this to superchainConfig?
    mapping(uint256 /* l2ChainId */ => bytes32) public releases; // Current release for each chain.

    // TODO how to prevent squatting on l2ChainId's? allow them to be overwritten by owner?

    constructor(
        uint256 standardChainId,
        SystemConfig _standardSystemConfig,
        ProtocolVersions _standardProtocolVersions
    ) {
        standardSystemConfig = _standardSystemConfig;
        standardProtocolVersions = _standardProtocolVersions;

        address standardPortal = standardSystemConfig.optimismPortal();
        standardSuperchainConfig = OptimismPortal(payable(standardPortal)).superchainConfig();
        ProxyAdmin standardProxyAdmin = ProxyAdmin(Proxy(payable(standardPortal)).admin());
        standardAddressManager = standardProxyAdmin.addressManager();

        register(standardChainId, latestRelease, standardSystemConfig, standardProtocolVersions);

        // _disableInitializers(); // TODO Uncomment once this is behind a proxy in the tests.
    }

    function initialize(address _owner) public initializer {
        __Ownable_init();
        transferOwnership(_owner);
    }

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

    function register(
        uint256 l2ChainId,
        bytes32 release_,
        SystemConfig systemConfig,
        ProtocolVersions pv
    )
        public
        onlyOwner
    {
        systemConfigs[l2ChainId] = systemConfig;
        protocolVersions[l2ChainId] = pv;
        releases[l2ChainId] = release_;
    }

    // This function signature is not expected to stay backwards compatible, as
    // initializers and required inputs may change.
    function deploy(uint256 l2ChainId, address proxyAdminOwner) public {
        require(address(systemConfigs[l2ChainId]) == address(0), "OpStackManager: Already deployed");

        // Deploy and partly configure the ProxyAdmin, temporarily making this contract the owner.
        ProxyAdmin proxyAdmin = new ProxyAdmin({ _owner: address(this) });
        proxyAdmin.setAddressManager(standardAddressManager);

        // Define proxies to be deployed.
        Proxies memory proxies;

        // Deploy ERC-1967 proxied contracts.
        proxies.l1ERC721Bridge = address(new Proxy(address(proxyAdmin)));
        proxies.l2OutputOracle = address(new Proxy(address(proxyAdmin)));
        proxies.optimismPortal = address(new Proxy(address(proxyAdmin)));
        proxies.systemConfig = address(new Proxy(address(proxyAdmin)));
        proxies.optimismMintableERC20Factory = address(new Proxy(address(proxyAdmin)));

        // Deploy legacy proxied contracts.
        proxies.l1StandardBridge = address(new L1ChugSplashProxy(address(proxyAdmin)));
        proxyAdmin.setProxyType(proxies.l1StandardBridge, ProxyAdmin.ProxyType.CHUGSPLASH);

        string memory contractName = "OVM_L1CrossDomainMessenger";
        proxies.l1CrossDomainMessenger = address(new ResolvedDelegateProxy(standardAddressManager, contractName));
        proxyAdmin.setProxyType(proxies.l1CrossDomainMessenger, ProxyAdmin.ProxyType.RESOLVED);
        proxyAdmin.setImplementationName(proxies.l1CrossDomainMessenger, contractName);

        // Initialize proxies.
        Implementation storage impl;
        bytes memory initdata;

        impl = _getLatestImplementation("L1ERC721Bridge");
        initdata = abi.encodeWithSelector(impl.initializer, proxies.l1CrossDomainMessenger, standardSuperchainConfig);
        proxyAdmin.upgradeAndCall(payable(proxies.l1ERC721Bridge), impl.logic, initdata);

        impl = _getLatestImplementation("L2OutputOracle");
        initdata = abi.encodeWithSelector(impl.initializer); // TODO
        proxyAdmin.upgradeAndCall(payable(proxies.l2OutputOracle), impl.logic, initdata);

        impl = _getLatestImplementation("OptimismPortal");
        initdata =
            abi.encodeWithSelector(impl.initializer, proxies.l2OutputOracle, proxies.systemConfig, standardSystemConfig);
        proxyAdmin.upgradeAndCall(payable(proxies.optimismPortal), impl.logic, initdata);

        impl = _getLatestImplementation("SystemConfig");
        initdata = abi.encodeWithSelector(impl.initializer); // TODO
        proxyAdmin.upgradeAndCall(payable(proxies.systemConfig), impl.logic, initdata);

        impl = _getLatestImplementation("OptimismMintableERC20Factory");
        initdata = abi.encodeWithSelector(impl.initializer, proxies.l1StandardBridge);
        proxyAdmin.upgradeAndCall(payable(proxies.optimismMintableERC20Factory), impl.logic, initdata);

        impl = _getLatestImplementation("L1CrossDomainMessenger");
        initdata = abi.encodeWithSelector(impl.initializer, standardSuperchainConfig, proxies.optimismPortal);
        proxyAdmin.upgradeAndCall(payable(proxies.l1CrossDomainMessenger), impl.logic, initdata);

        impl = _getLatestImplementation("L1CrossDomainMessenger");
        initdata = abi.encodeWithSelector(impl.initializer, standardSuperchainConfig, proxies.optimismPortal);
        proxyAdmin.upgradeAndCall(payable(proxies.l1CrossDomainMessenger), impl.logic, initdata);

        // Now that deployment is complete, we transfer ownership to the specified owner.
        proxyAdmin.transferOwnership(proxyAdminOwner);

        // Save off this deploy.
        register(l2ChainId, latestRelease, SystemConfig(proxies.systemConfig), standardProtocolVersions);
    }

    function upgrade(uint256 l2ChainId) public {
        // TODO also take initializer data as input args.
        // This method is bespoke for `latestRelease` so changes with upgrades to this contract.
        // Methods like `upgrade_v1_3_0` should be added for older releases.
    }

    function _getLatestImplementation(bytes32 name) internal view returns (Implementation storage impl) {
        impl = implementations[latestRelease][name];
    }
}

/// @dev https://eips.ethereum.org/EIPS/eip-5202
/// @dev This approach is probably overkill after MCP, just was playing around with the approach here.
contract BlueprintStuff {
    struct Instance {
        bytes32 name;
        address owner;
        address systemConfig;
    }

    struct Implementation {
        address blueprint; // Address containing the initcode.
        bytes4 initializer; // Function selector for the initializer.
    }

    struct BlueprintPreamble {
        uint256 ercVersion;
        bytes preambleData;
        bytes initcode;
    }

    mapping(uint256 /* chainId */ => Instance) public instances;
    mapping(bytes32 /* version */ => mapping(bytes32 /* name */ => Implementation)) public implementations;

    bytes32[] public contracts = [
        bytes32("SystemConfig"),
        bytes32("OptimismPortal"),
        bytes32("OptimismMintableERC20Factory"),
        bytes32("L1CrossDomainMessenger"),
        bytes32("L1StandardBridge"),
        bytes32("L1ERC721Bridge"),
        bytes32("L2OutputOracle")
    ];

    function deploy(uint256 chainId, bytes32 name, address _owner, bytes32 version) public {
        require(instances[chainId].systemConfig == address(0), "OpStackManager: Instance already exists");
        address systemConfig;
        for (uint256 i = 0; i < contracts.length; i++) {
            Implementation storage impl = implementations[version][contracts[i]];
            require(impl.blueprint != address(0), "OpStackManager: Implementation not found");

            // Deploy and initialize the blueprint.
            address addr = deployBlueprint(impl.blueprint.code, bytes32(bytes20(_owner)));
            (bool ok,) = addr.call(abi.encodeWithSelector(impl.initializer));
            require(ok, "OpStackManager: Initialization failed");

            if (contracts[i] == bytes32("SystemConfig")) {
                systemConfig = addr;
            }
        }
        require(systemConfig != address(0), "OpStackManager: SystemConfig not found");
        instances[chainId] = Instance(name, _owner, systemConfig);
    }

    function parseBlueprintPreamble(bytes memory bytecode) internal pure returns (BlueprintPreamble memory) {
        require(bytecode[0] == 0xFE && bytecode[1] == 0x71, "OPStackManager: Not a blueprint!");

        uint256 ercVersion = (uint256(uint8(bytecode[2])) & 0xFC) >> 2;
        uint256 nLengthBytes = uint256(uint8(bytecode[2])) & 0x03;
        require(nLengthBytes != 0x03, "OPStackManager:Reserved bits are set");

        uint256 dataLength;
        if (nLengthBytes > 0) {
            require(bytecode.length >= 3 + nLengthBytes, "OPStackManager: Invalid blueprint bytecode length");
            bytes memory lengthBytes = new bytes(nLengthBytes);
            for (uint256 i = 0; i < nLengthBytes; i++) {
                lengthBytes[i] = bytecode[3 + i];
            }
            dataLength = bytesToUint(lengthBytes);
        }

        bytes memory preambleData;
        if (nLengthBytes > 0) {
            uint256 dataStart = 3 + nLengthBytes;
            require(bytecode.length >= dataStart + dataLength, "OPStackManager: Invalid blueprint bytecode length");
            preambleData = new bytes(dataLength);
            for (uint256 i = 0; i < dataLength; i++) {
                preambleData[i] = bytecode[dataStart + i];
            }
        }

        uint256 initcodeStart = 3 + nLengthBytes + dataLength;
        require(bytecode.length > initcodeStart, "OPStackManager: Empty initcode");
        bytes memory initcode = new bytes(bytecode.length - initcodeStart);
        for (uint256 i = 0; i < initcode.length; i++) {
            initcode[i] = bytecode[initcodeStart + i];
        }

        return BlueprintPreamble(ercVersion, preambleData, initcode);
    }

    function bytesToUint(bytes memory b) internal pure returns (uint256) {
        uint256 number;
        for (uint256 i = 0; i < b.length; i++) {
            number = number + uint256(uint8(b[i])) * (2 ** (8 * (b.length - (i + 1))));
        }
        return number;
    }

    function deployBlueprint(bytes memory bytecode, bytes32 salt) internal returns (address addr) {
        bytes memory initcode = parseBlueprintPreamble(bytecode).initcode;

        assembly {
            let ptr := mload(0x40)
            mstore(ptr, add(initcode, 0x20))
            mstore(ptr, mload(initcode))
            addr := create2(0, ptr, mload(ptr), salt)
        }

        require(addr != address(0), "OPStackManager: Blueprint deployment failed");
    }
}
