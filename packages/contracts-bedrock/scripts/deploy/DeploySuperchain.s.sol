// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

// For all broadcasts in this script we explicitly specify the deployer as `msg.sender` because for
// testing we deploy this script from a test contract. If we provide no argument, the foundry
// default sender would be the broadcaster during test, but the broadcaster needs to be the deployer
// since they are set to the initial proxy admin owner.
contract DeploySuperchain is Script {
    struct Input {
        // Role inputs.
        address guardian;
        address superchainProxyAdminOwner;
        // Other inputs.
        bool paused;
    }

    struct Output {
        ISuperchainConfig superchainConfigImpl;
        ISuperchainConfig superchainConfigProxy;
        IProxyAdmin superchainProxyAdmin;
    }

    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    // -------- Core Deployment Methods --------

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }

    function run(Input memory _input) public returns (Output memory output_) {
        // Make sure the inputs are all set
        assertValidInput(_input);

        // Deploy the proxy admin, with the owner set to the deployer.
        deploySuperchainProxyAdmin(_input, output_);

        // Deploy and initialize the superchain contracts.
        deploySuperchainImplementationContracts(_input, output_);
        deployAndInitializeSuperchainConfig(_input, output_);

        // Transfer ownership of the ProxyAdmin from the deployer to the specified owner.
        transferProxyAdminOwnership(_input, output_);

        // Output assertions, to make sure outputs were assigned correctly.
        assertValidOutput(_input, output_);
    }

    // -------- Deployment Steps --------

    function deploySuperchainProxyAdmin(Input memory, Output memory _output) private {
        // Deploy the proxy admin, with the owner set to the deployer.
        // We explicitly specify the deployer as `msg.sender` because for testing we deploy this script from a test
        // contract. If we provide no argument, the foundry default sender would be the broadcaster during test, but the
        // broadcaster needs to be the deployer since they are set to the initial proxy admin owner.
        vm.broadcast(msg.sender);
        IProxyAdmin superchainProxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (msg.sender)))
            })
        );

        vm.label(address(superchainProxyAdmin), "SuperchainProxyAdmin");
        _output.superchainProxyAdmin = superchainProxyAdmin;
    }

    function deploySuperchainImplementationContracts(Input memory, Output memory _output) private {
        // Deploy implementation contracts.
        ISuperchainConfig superchainConfigImpl = ISuperchainConfig(
            DeployUtils.createDeterministic({
                _name: "SuperchainConfig",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperchainConfig.__constructor__, ())),
                _salt: _salt
            })
        );

        vm.label(address(superchainConfigImpl), "SuperchainConfigImpl");

        _output.superchainConfigImpl = superchainConfigImpl;
    }

    function deployAndInitializeSuperchainConfig(Input memory _input, Output memory _output) private {
        address guardian = _input.guardian;

        IProxyAdmin superchainProxyAdmin = _output.superchainProxyAdmin;
        ISuperchainConfig superchainConfigImpl = _output.superchainConfigImpl;

        vm.startBroadcast(msg.sender);
        ISuperchainConfig superchainConfigProxy = ISuperchainConfig(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IProxy.__constructor__, (address(superchainProxyAdmin)))
                )
            })
        );
        superchainProxyAdmin.upgradeAndCall(
            payable(address(superchainConfigProxy)),
            address(superchainConfigImpl),
            abi.encodeCall(ISuperchainConfig.initialize, (guardian))
        );
        vm.stopBroadcast();

        vm.label(address(superchainConfigProxy), "SuperchainConfigProxy");
        _output.superchainConfigProxy = superchainConfigProxy;
    }

    function transferProxyAdminOwnership(Input memory _input, Output memory _output) private {
        address superchainProxyAdminOwner = _input.superchainProxyAdminOwner;

        IProxyAdmin superchainProxyAdmin = _output.superchainProxyAdmin;
        DeployUtils.assertValidContractAddress(address(superchainProxyAdmin));

        vm.broadcast(msg.sender);
        superchainProxyAdmin.transferOwnership(superchainProxyAdminOwner);
    }

    function assertValidInput(Input memory _input) internal pure {
        require(_input.guardian != address(0), "DeploySuperchain: guardian not set");
        require(_input.superchainProxyAdminOwner != address(0), "DeploySuperchain: superchainProxyAdminOwner not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) public {
        assertValidContractAddresses(_input, _output);
        assertValidSuperchainProxyAdmin(_input, _output);
        assertValidSuperchainConfig(_input, _output);
    }

    function assertValidContractAddresses(Input memory, Output memory _output) internal {
        address[] memory addrs = Solarray.addresses(
            address(_output.superchainProxyAdmin),
            address(_output.superchainConfigImpl),
            address(_output.superchainConfigProxy)
        );
        DeployUtils.assertValidContractAddresses(addrs);

        // To read the implementations we prank as the zero address due to the proxyCallIfNotAdmin modifier.
        vm.startPrank(address(0));
        address actualSuperchainConfigImpl = IProxy(payable(address(_output.superchainConfigProxy))).implementation();
        vm.stopPrank();

        require(actualSuperchainConfigImpl == address(_output.superchainConfigImpl), "100"); // nosemgrep:
            // sol-style-malformed-require
    }

    function assertValidSuperchainProxyAdmin(Input memory _input, Output memory _output) internal view {
        require(_output.superchainProxyAdmin.owner() == _input.superchainProxyAdminOwner, "SPA-10");
    }

    function assertValidSuperchainConfig(Input memory _input, Output memory _output) internal {
        // Proxy checks.
        ISuperchainConfig superchainConfig = _output.superchainConfigProxy;
        DeployUtils.assertInitialized({
            _contractAddress: address(superchainConfig),
            _isProxy: true,
            _slot: 0,
            _offset: 0
        });
        require(superchainConfig.guardian() == _input.guardian, "SUPCON-10");

        vm.startPrank(address(0));
        require(
            IProxy(payable(address(superchainConfig))).implementation() == address(_output.superchainConfigImpl),
            "SUPCON-30"
        );
        require(
            IProxy(payable(address(superchainConfig))).admin() == address(_output.superchainProxyAdmin), "SUPCON-40"
        );
        vm.stopPrank();

        // Implementation checks
        superchainConfig = _output.superchainConfigImpl;
        require(superchainConfig.guardian() == address(0), "SUPCON-50");
    }
}
