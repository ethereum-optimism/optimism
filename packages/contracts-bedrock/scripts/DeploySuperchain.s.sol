// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

contract DeploySuperchainInput {
    struct Roles {
        address proxyAdminOwner;
        address protocolVersionsOwner;
        address guardian;
    }

    struct Input {
        Roles roles;
        bool paused;
        ProtocolVersion requiredProtocolVersion;
        ProtocolVersion recommendedProtocolVersion;
    }

    // The data from the input struct gets stored in individual variables to make them accessible in
    // a stronger type-safe manner (as opposed to a single struct getter where the caller may
    // inadvertently transpose two addresses, for example) without the need for defining getters for
    // each field.
    bool internal inputSet = false;
    Input internal input;

    address public proxyAdminOwner;
    address public protocolVersionsOwner;
    address public guardian;
    bool public paused;
    ProtocolVersion public requiredProtocolVersion;
    ProtocolVersion public recommendedProtocolVersion;

    function loadInput(string memory _infile) public {
        _infile;
        Input memory parsedInput; // TODO parse the input file.
        loadInput(parsedInput);
        require(false, "DeploySuperchainInput: loadInput is not implemented");
    }

    function loadInput(Input memory _input) public {
        require(!inputSet, "DeploySuperchainInput: input already set");

        // All assertions on inputs happen here. You cannot set any inputs unless they're all valid.
        require(input.roles.proxyAdminOwner != address(0), "DeploySuperchainInput: empty proxyAdminOwner");
        require(input.roles.protocolVersionsOwner != address(0), "DeploySuperchainInput: empty protocolVersionsOwner");
        require(input.roles.guardian != address(0), "DeploySuperchainInput: empty guardian");

        inputSet = true;
        input = _input;

        proxyAdminOwner = _input.roles.proxyAdminOwner;
        protocolVersionsOwner = _input.roles.protocolVersionsOwner;
        guardian = _input.roles.guardian;
        paused = _input.paused;
        requiredProtocolVersion = _input.requiredProtocolVersion;
        recommendedProtocolVersion = _input.recommendedProtocolVersion;
    }

    function inputs() public view returns (Input memory) {
        require(inputSet, "DeploySuperchainInput: input not set");
        return input;
    }
}

contract DeploySuperchainOutput {
    struct Output {
        ProxyAdmin superchainProxyAdmin;
        SuperchainConfig superchainConfigImpl;
        SuperchainConfig superchainConfigProxy;
        ProtocolVersions protocolVersionsImpl;
        ProtocolVersions protocolVersionsProxy;
    }

    // The data from the output struct gets stored in individual variables to make them accessible
    // in a stronger type-safe manner (as opposed to a single struct getter where the caller may
    // inadvertently transpose two addresses, for example) without the need for defining getters for
    // each field.
    bool internal outputSet = false;
    Output internal output;

    ProxyAdmin public superchainProxyAdmin;
    SuperchainConfig public superchainConfigImpl;
    SuperchainConfig public superchainConfigProxy;
    ProtocolVersions public protocolVersionsImpl;
    ProtocolVersions public protocolVersionsProxy;

    function saveOutput(Output memory _output) public {
        require(!outputSet, "DeploySuperchainOutput: output already set");
        outputSet = true;

        output = _output;
        superchainProxyAdmin = _output.superchainProxyAdmin;
        superchainConfigImpl = _output.superchainConfigImpl;
        superchainConfigProxy = _output.superchainConfigProxy;
        protocolVersionsImpl = _output.protocolVersionsImpl;
        protocolVersionsProxy = _output.protocolVersionsProxy;
    }

    function writeOutput(string memory _outfile) public view {
        require(outputSet, "DeploySuperchainOutput: output not set");
        _outfile; // TODO: write to file.
        require(false, "DeploySuperchainOutput: saveOutput not implemented");
    }

    function outputs() public view returns (Output memory) {
        require(outputSet, "DeploySuperchainOutput: output not set");
        return output;
    }
}

/// @notice Deploys the Superchain contracts that can be shared by many chains.
/// We intentionally use the terms "Input" and "Output" to clearly distinguish this script from the
/// existing ones that use terms of "Config" and "Artifacts".
contract DeploySuperchain is Script {
    function toAddress(address _sender, string memory _identifier) public pure returns (address) {
        return address(uint160(uint256(keccak256(abi.encode(_sender, _identifier)))));
    }

    /// @notice This entrypoint is for end-users to deploy from an input file and write to an output file.
    function run(string memory _infile) public {
        // Using fileIO, so deploy the IO helper contracts.
        DeploySuperchainInput dsi = DeploySuperchainInput(toAddress(msg.sender, "optimism.DeploySuperchainInput"));
        DeploySuperchainOutput dso = DeploySuperchainOutput(toAddress(msg.sender, "optimism.DeploySuperchainOutput"));
        vm.etch(address(dsi), type(DeploySuperchainInput).runtimeCode);
        vm.etch(address(dso), type(DeploySuperchainOutput).runtimeCode);

        // Parse the input file using the DeploySuperchainInput contract.
        dsi.loadInput(_infile);

        // Run the deployment script.
        // This also writes the output to the DeploySuperchainOutput contract.
        runWithoutIO(dsi, dso);

        // Request a file to be written with the output data.
        string memory outfile = "";
        dso.writeOutput(outfile);
        require(false, "DeploySuperchain: run is not implemented");
    }

    /// @notice This entrypoint is useful for e2e testing purposes, and doesn't use any file I/O.
    function runWithoutIO(DeploySuperchainInput _dsi, DeploySuperchainOutput _dso) public {
        // Retrieve the input data from the input contract. This reverts if the input is not set.
        DeploySuperchainInput.Input memory input = _dsi.inputs();

        // Initialize the output struct.
        DeploySuperchainOutput.Output memory output;

        // Deploy the proxy admin, with the owner set to the deployer.
        // We explicitly specify the deployer as `msg.sender` because for testing we deploy this script from a test
        // contract. If we provide no argument, the foundry default sender is be the broadcaster during test, but the
        // broadcaster needs to be the deployer since they are set to the initial proxy admin owner.
        vm.startBroadcast(msg.sender);

        output.superchainProxyAdmin = new ProxyAdmin(msg.sender);
        vm.label(address(output.superchainProxyAdmin), "SuperchainProxyAdmin");

        // Deploy implementation contracts.
        output.superchainConfigImpl = new SuperchainConfig();
        output.protocolVersionsImpl = new ProtocolVersions();

        // Deploy and initialize the proxies.
        output.superchainConfigProxy = SuperchainConfig(address(new Proxy(address(output.superchainProxyAdmin))));
        vm.label(address(output.superchainConfigProxy), "SuperchainConfigProxy");
        output.superchainProxyAdmin.upgradeAndCall(
            payable(address(output.superchainConfigProxy)),
            address(output.superchainConfigImpl),
            abi.encodeCall(SuperchainConfig.initialize, (input.roles.guardian, input.paused))
        );

        output.protocolVersionsProxy = ProtocolVersions(address(new Proxy(address(output.superchainProxyAdmin))));
        vm.label(address(output.protocolVersionsProxy), "ProtocolVersionsProxy");
        output.superchainProxyAdmin.upgradeAndCall(
            payable(address(output.protocolVersionsProxy)),
            address(output.protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (input.roles.protocolVersionsOwner, input.requiredProtocolVersion, input.recommendedProtocolVersion)
            )
        );

        // Transfer ownership of the ProxyAdmin from the deployer to the specified owner.
        output.superchainProxyAdmin.transferOwnership(input.roles.proxyAdminOwner);

        vm.stopBroadcast();

        // Output assertions, to make sure outputs were assigned correctly.
        address[] memory addresses = new address[](5);
        addresses[0] = address(output.superchainProxyAdmin);
        addresses[1] = address(output.superchainConfigImpl);
        addresses[2] = address(output.superchainConfigProxy);
        addresses[3] = address(output.protocolVersionsImpl);
        addresses[4] = address(output.protocolVersionsProxy);

        for (uint256 i = 0; i < addresses.length; i++) {
            require(addresses[i] != address(0), string.concat("zero address at index ", vm.toString(i)));
            require(addresses[i].code.length > 0, string.concat("no code at index ", vm.toString(i)));
        }

        // All addresses should be unique.
        for (uint256 i = 0; i < addresses.length; i++) {
            for (uint256 j = i + 1; j < addresses.length; j++) {
                string memory err = string.concat("duplicates at: ", vm.toString(i), ",", vm.toString(j));
                require(addresses[i] != addresses[j], err);
            }
        }

        // Deploy successful, save off output.
        _dso.saveOutput(output);
    }
}
