// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";

import { Deployer } from "./Deployer.sol";
import { PeripheryDeployConfig } from "./PeripheryDeployConfig.s.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

import { Faucet } from "src/periphery/faucet/Faucet.sol";

/// @title DeployPeriphery
/// @notice Script used to deploy periphery contracts.
contract DeployPeriphery is Deployer {
    PeripheryDeployConfig cfg;

    /// @notice The name of the script, used to ensure the right deploy artifacts
    ///         are used.
    function name() public pure override returns (string memory) {
        return "DeployPeriphery";
    }

    function setUp() public override {
        super.setUp();

        string memory path = string.concat(vm.projectRoot(), "/periphery-deploy-config/", deploymentContext, ".json");
        cfg = new PeripheryDeployConfig(path);

        console.log("Deploying from %s", deployScript);
        console.log("Deployment context: %s", deploymentContext);
    }

    /// @notice Deploy all of the periphery contracts
    function run() public {
        console.log("Deploying all periphery contracts");

        deployProxies();
        deployImplementations();

        initializeFaucet();
    }

    /// @notice Deploy all of the proxies
    function deployProxies() public {
        deployProxyAdmin();

        deployFaucetProxy();
    }

    /// @notice Deploy all of the implementations
    function deployImplementations() public {
        deployFaucet();
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast();
        _;
        vm.stopBroadcast();
    }

    /// @notice Deploy the ProxyAdmin
    function deployProxyAdmin() public broadcast returns (address addr_) {
        bytes32 salt = keccak256(bytes("ProxyAdmin"));
        bytes32 initCodeHash = keccak256(abi.encodePacked(type(ProxyAdmin).creationCode, abi.encode(msg.sender)));
        address preComputedAddress = computeCreate2Address(salt, initCodeHash);
        if (preComputedAddress.code.length > 0) {
            console.log("ProxyAdmin already deployed at %s", preComputedAddress);
            save("ProxyAdmin", preComputedAddress);
            addr_ = preComputedAddress;
        } else {
            ProxyAdmin admin = new ProxyAdmin{ salt: salt }({
              _owner: msg.sender
            });
            require(admin.owner() == msg.sender);

            save("ProxyAdmin", address(admin));
            console.log("ProxyAdmin deployed at %s", address(admin));

            addr_ = address(admin);
        }
    }

    /// @notice Deploy the FaucetProxy
    function deployFaucetProxy() public broadcast returns (address addr_) {
        bytes32 salt = keccak256(bytes("FaucetProxy"));
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        bytes32 initCodeHash = keccak256(abi.encodePacked(type(Proxy).creationCode, abi.encode(proxyAdmin)));
        address preComputedAddress = computeCreate2Address(salt, initCodeHash);
        if (preComputedAddress.code.length > 0) {
            console.log("FaucetProxy already deployed at %s", preComputedAddress);
            save("FaucetProxy", preComputedAddress);
            addr_ = preComputedAddress;
        } else {
            Proxy proxy = new Proxy{ salt: salt }({
              _admin: proxyAdmin
            });
            address admin = address(uint160(uint256(vm.load(address(proxy), OWNER_KEY))));
            require(admin == proxyAdmin);

            save("FaucetProxy", address(proxy));
            console.log("FaucetProxy deployed at %s", address(proxy));

            addr_ = address(proxy);
        }
    }

    /// @notice Deploy the faucet contract.
    function deployFaucet() public broadcast returns (address addr_) {
        bytes32 salt = keccak256(bytes("Faucet"));
        bytes32 initCodeHash = keccak256(abi.encodePacked(type(Faucet).creationCode, abi.encode(cfg.faucetAdmin())));
        address preComputedAddress = computeCreate2Address(salt, initCodeHash);
        if (preComputedAddress.code.length > 0) {
            console.log("Faucet already deployed at %s", preComputedAddress);
            save("Faucet", preComputedAddress);
            addr_ = preComputedAddress;
        } else {
            Faucet faucet = new Faucet{ salt: salt }(cfg.faucetAdmin());
            require(faucet.ADMIN() == cfg.faucetAdmin());

            save("Faucet", address(faucet));
            console.log("Faucet deployed at %s", address(faucet));

            addr_ = address(faucet);
        }
    }

    /// @notice Initialize the Faucet
    function initializeFaucet() public broadcast {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address faucetProxy = mustGetAddress("FaucetProxy");
        address faucet = mustGetAddress("Faucet");
        address implementationAddress = proxyAdmin.getProxyImplementation(faucetProxy);
        if (implementationAddress == faucet) {
            console.log("Faucet proxy implementation already set");
        } else {
            proxyAdmin.upgrade({ _proxy: payable(faucetProxy), _implementation: faucet });
        }

        require(Faucet(payable(faucetProxy)).ADMIN() == Faucet(payable(faucet)).ADMIN());
    }
}
