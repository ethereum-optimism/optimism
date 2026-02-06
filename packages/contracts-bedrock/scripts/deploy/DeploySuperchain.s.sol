// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions, ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
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
        address protocolVersionsOwner;
        address superchainProxyAdminOwner;
        // Other inputs.
        bool paused;
        bytes32 recommendedProtocolVersion;
        bytes32 requiredProtocolVersion;
    }

    /// @notice InternalInput is created based on Input by converting the bytes32 protocol versions to ProtocolVersion
    /// types
    //
    // ProtocolVersion type is based on uint256 which conflicts with downstream types (like e.g. ProtocolVersion from
    // op-geth)
    // so to keep the ABI externally compatible, we expose it simply as bytes32
    struct InternalInput {
        // Role inputs.
        address guardian;
        address protocolVersionsOwner;
        address superchainProxyAdminOwner;
        // Other inputs.
        bool paused;
        ProtocolVersion recommendedProtocolVersion;
        ProtocolVersion requiredProtocolVersion;
    }

    struct Output {
        IProtocolVersions protocolVersionsImpl;
        IProtocolVersions protocolVersionsProxy;
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
        // Convert the external Input to InternalInput
        InternalInput memory internalInput = toInternalInput(_input);

        // Make sure the inputs are all set
        assertValidInput(internalInput);

        // Deploy the proxy admin, with the owner set to the deployer.
        deploySuperchainProxyAdmin(internalInput, output_);

        // Deploy and initialize the superchain contracts.
        deploySuperchainImplementationContracts(internalInput, output_);
        deployAndInitializeSuperchainConfig(internalInput, output_);
        deployAndInitializeProtocolVersions(internalInput, output_);

        // Transfer ownership of the ProxyAdmin from the deployer to the specified owner.
        transferProxyAdminOwnership(internalInput, output_);

        // Output assertions, to make sure outputs were assigned correctly.
        assertValidOutput(internalInput, output_);
    }

    // -------- Deployment Steps --------

    function deploySuperchainProxyAdmin(InternalInput memory, Output memory _output) private {
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

    function deploySuperchainImplementationContracts(InternalInput memory, Output memory _output) private {
        // Deploy implementation contracts.
        ISuperchainConfig superchainConfigImpl = ISuperchainConfig(
            DeployUtils.createDeterministic({
                _name: "SuperchainConfig",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperchainConfig.__constructor__, ())),
                _salt: _salt
            })
        );
        IProtocolVersions protocolVersionsImpl = IProtocolVersions(
            DeployUtils.createDeterministic({
                _name: "ProtocolVersions",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProtocolVersions.__constructor__, ())),
                _salt: _salt
            })
        );

        vm.label(address(superchainConfigImpl), "SuperchainConfigImpl");
        vm.label(address(protocolVersionsImpl), "ProtocolVersionsImpl");

        _output.superchainConfigImpl = superchainConfigImpl;
        _output.protocolVersionsImpl = protocolVersionsImpl;
    }

    function deployAndInitializeSuperchainConfig(InternalInput memory _input, Output memory _output) private {
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

    function deployAndInitializeProtocolVersions(InternalInput memory _input, Output memory _output) private {
        address protocolVersionsOwner = _input.protocolVersionsOwner;
        ProtocolVersion requiredProtocolVersion = _input.requiredProtocolVersion;
        ProtocolVersion recommendedProtocolVersion = _input.recommendedProtocolVersion;

        IProxyAdmin superchainProxyAdmin = _output.superchainProxyAdmin;
        IProtocolVersions protocolVersionsImpl = _output.protocolVersionsImpl;

        vm.startBroadcast(msg.sender);
        IProtocolVersions protocolVersionsProxy = IProtocolVersions(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IProxy.__constructor__, (address(superchainProxyAdmin)))
                )
            })
        );
        superchainProxyAdmin.upgradeAndCall(
            payable(address(protocolVersionsProxy)),
            address(protocolVersionsImpl),
            abi.encodeCall(
                IProtocolVersions.initialize,
                (protocolVersionsOwner, requiredProtocolVersion, recommendedProtocolVersion)
            )
        );
        vm.stopBroadcast();

        vm.label(address(protocolVersionsProxy), "ProtocolVersionsProxy");
        _output.protocolVersionsProxy = protocolVersionsProxy;
    }

    function transferProxyAdminOwnership(InternalInput memory _input, Output memory _output) private {
        address superchainProxyAdminOwner = _input.superchainProxyAdminOwner;

        IProxyAdmin superchainProxyAdmin = _output.superchainProxyAdmin;
        DeployUtils.assertValidContractAddress(address(superchainProxyAdmin));

        vm.broadcast(msg.sender);
        superchainProxyAdmin.transferOwnership(superchainProxyAdminOwner);
    }

    function assertValidInput(InternalInput memory _input) internal pure {
        require(_input.guardian != address(0), "DeploySuperchain: guardian not set");
        require(_input.protocolVersionsOwner != address(0), "DeploySuperchain: protocolVersionsOwner not set");
        require(
            ProtocolVersion.unwrap(_input.requiredProtocolVersion) != 0,
            "DeploySuperchain: requiredProtocolVersion not set"
        );
        require(
            ProtocolVersion.unwrap(_input.recommendedProtocolVersion) != 0,
            "DeploySuperchain: recommendedProtocolVersion not set"
        );
        require(_input.superchainProxyAdminOwner != address(0), "DeploySuperchain: superchainProxyAdminOwner not set");
    }

    function assertValidOutput(InternalInput memory _input, Output memory _output) public {
        assertValidContractAddresses(_input, _output);
        assertValidSuperchainProxyAdmin(_input, _output);
        assertValidSuperchainConfig(_input, _output);
        assertValidProtocolVersions(_input, _output);
    }

    function assertValidContractAddresses(InternalInput memory, Output memory _output) internal {
        address[] memory addrs = Solarray.addresses(
            address(_output.superchainProxyAdmin),
            address(_output.superchainConfigImpl),
            address(_output.superchainConfigProxy),
            address(_output.protocolVersionsImpl),
            address(_output.protocolVersionsProxy)
        );
        DeployUtils.assertValidContractAddresses(addrs);

        // To read the implementations we prank as the zero address due to the proxyCallIfNotAdmin modifier.
        vm.startPrank(address(0));
        address actualSuperchainConfigImpl = IProxy(payable(address(_output.superchainConfigProxy))).implementation();
        address actualProtocolVersionsImpl = IProxy(payable(address(_output.protocolVersionsProxy))).implementation();
        vm.stopPrank();

        require(actualSuperchainConfigImpl == address(_output.superchainConfigImpl), "100"); // nosemgrep:
        // sol-style-malformed-require
        require(actualProtocolVersionsImpl == address(_output.protocolVersionsImpl), "200"); // nosemgrep:
        // sol-style-malformed-require
    }

    function assertValidSuperchainProxyAdmin(InternalInput memory _input, Output memory _output) internal view {
        require(_output.superchainProxyAdmin.owner() == _input.superchainProxyAdminOwner, "SPA-10");
    }

    function assertValidSuperchainConfig(InternalInput memory _input, Output memory _output) internal {
        // Proxy checks.
        ISuperchainConfig superchainConfig = _output.superchainConfigProxy;
        DeployUtils.assertInitialized({
            _contractAddress: address(superchainConfig), _isProxy: true, _slot: 0, _offset: 0
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

    function assertValidProtocolVersions(InternalInput memory _input, Output memory _output) internal {
        // Proxy checks.
        IProtocolVersions pv = _output.protocolVersionsProxy;
        DeployUtils.assertInitialized({ _contractAddress: address(pv), _isProxy: true, _slot: 0, _offset: 0 });
        require(pv.owner() == _input.protocolVersionsOwner, "PV-10");
        require(
            ProtocolVersion.unwrap(pv.required()) == ProtocolVersion.unwrap(_input.requiredProtocolVersion), "PV-20"
        );
        require(
            ProtocolVersion.unwrap(pv.recommended()) == ProtocolVersion.unwrap(_input.recommendedProtocolVersion),
            "PV-30"
        );

        vm.startPrank(address(0));
        require(IProxy(payable(address(pv))).implementation() == address(_output.protocolVersionsImpl), "PV-40");
        require(IProxy(payable(address(pv))).admin() == address(_output.superchainProxyAdmin), "PV-50");
        vm.stopPrank();

        // Implementation checks.
        pv = _output.protocolVersionsImpl;
        require(pv.owner() == address(0), "PV-60");
        require(ProtocolVersion.unwrap(pv.required()) == 0, "PV-70");
        require(ProtocolVersion.unwrap(pv.recommended()) == 0, "PV-80");
    }

    function toInternalInput(Input memory _input) internal pure returns (InternalInput memory input_) {
        input_ = InternalInput(
            _input.guardian,
            _input.protocolVersionsOwner,
            _input.superchainProxyAdminOwner,
            _input.paused,
            ProtocolVersion.wrap(uint256(_input.recommendedProtocolVersion)),
            ProtocolVersion.wrap(uint256(_input.requiredProtocolVersion))
        );
    }
}
