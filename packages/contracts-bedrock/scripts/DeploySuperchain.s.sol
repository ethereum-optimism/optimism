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

    Input internal input;
    bool internal inputSet = false;

    // -------- Load inputs --------

    function loadInput(string memory _infile) public {
        _infile;
        Input memory parsedInput; // TODO parse the input file.
        loadInput(parsedInput);
        require(false, "DeploySuperchainInput: loadInput is not implemented");
    }

    function loadInput(Input memory _input) public {
        require(!inputSet, "DeploySuperchainInput: input already set");
        input = _input;
    }

    // -------- Getter methods for inputs --------

    function inputs() public view returns (Input memory) {
        require(inputSet, "DeploySuperchainInput: input not set");
        return input;
    }

    function proxyAdminOwner() public view returns (address out) {
        out = input.roles.proxyAdminOwner;
        require(out != address(0), "DeploySuperchainInput: proxyAdminOwner not set");
    }

    function protocolVersionsOwner() public view returns (address out) {
        out = input.roles.protocolVersionsOwner;
        require(out != address(0), "DeploySuperchainInput: protocolVersionsOwner not set");
    }

    function guardian() public view returns (address out) {
        out = input.roles.guardian;
        require(out != address(0), "DeploySuperchainInput: guardian not set");
    }

    function paused() public view returns (bool out) {
        out = input.paused;
    }

    function requiredProtocolVersion() public view returns (ProtocolVersion out) {
        out = input.requiredProtocolVersion;
        require(ProtocolVersion.unwrap(out) != 0, "DeploySuperchainInput: requiredProtocolVersion not set");
    }

    function recommendedProtocolVersion() public view returns (ProtocolVersion out) {
        out = input.recommendedProtocolVersion;
        require(ProtocolVersion.unwrap(out) != 0, "DeploySuperchainInput: recommendedProtocolVersion not set");
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

    Output internal output;
    bool internal outputSet = false;

    // -------- Save outputs --------

    function saveOutput(Output memory _output) public {
        require(!outputSet, "DeploySuperchainOutput: output already set");
        output = _output;
        outputSet = true;
    }

    function writeOutput(string memory _outfile) public view {
        require(outputSet, "DeploySuperchainOutput: output not set");
        _outfile; // TODO: write to file.
        require(false, "DeploySuperchainOutput: saveOutput not implemented");
    }

    // -------- Getter methods for outputs --------
    function outputs() public view returns (Output memory) {
        require(outputSet, "DeploySuperchainOutput: output not set");
        return output;
    }

    function superchainProxyAdmin() public view returns (ProxyAdmin out) {
        out = output.superchainProxyAdmin;
        require(address(out) != address(0), "DeploySuperchainOutput: superchainProxyAdmin not set");
    }

    function superchainConfigImpl() public view returns (SuperchainConfig out) {
        out = output.superchainConfigImpl;
        require(address(out) != address(0), "DeploySuperchainOutput: superchainConfigImpl not set");
    }

    function superchainConfigProxy() public view returns (SuperchainConfig out) {
        out = output.superchainConfigProxy;
        require(address(out) != address(0), "DeploySuperchainOutput: superchainConfigProxy not set");
    }

    function protocolVersionsImpl() public view returns (ProtocolVersions out) {
        out = output.protocolVersionsImpl;
        require(address(out) != address(0), "DeploySuperchainOutput: protocolVersionsImpl not set");
    }

    function protocolVersionsProxy() public view returns (ProtocolVersions out) {
        out = output.protocolVersionsProxy;
        require(address(out) != address(0), "DeploySuperchainOutput: protocolVersionsProxy not set");
    }
}

/// @notice Deploys the Superchain contracts that can be shared by many chains.
/// We intentionally use the terms "Input" and "Output" to clearly distinguish this script from the
/// existing ones that use terms of "Config" and "Artifacts".
contract DeploySuperchain is Script {
    DeploySuperchainInput dsi;
    DeploySuperchainOutput dso;

    function toAddress(address _sender, string memory _identifier) public pure returns (address) {
        return address(uint160(uint256(keccak256(abi.encode(_sender, _identifier)))));
    }

    function init() internal {
        // Deploy the input and output contracts.
        // This function is a no-op on subsequent calls.
        if (address(dsi) == address(0)) {
            dsi = DeploySuperchainInput(toAddress(msg.sender, "optimism.DeploySuperchainInput"));
            dso = DeploySuperchainOutput(toAddress(msg.sender, "optimism.DeploySuperchainOutput"));
            vm.etch(address(dsi), type(DeploySuperchainInput).runtimeCode);
            vm.etch(address(dso), type(DeploySuperchainOutput).runtimeCode);
        }
    }

    /// @notice This entrypoint is for end-users to deploy from an input file and write to an output file.
    function run(string memory _infile) public {
        init();
        dsi.loadInput(_infile);

        runWithoutIO(dsi.inputs());

        string memory outfile = "";
        dso.writeOutput(outfile);
        require(false, "DeploySuperchain: run is not implemented");
    }

    /// @notice This entrypoint is useful for e2e testing purposes, and doesn't use any file I/O.
    function runWithoutIO(DeploySuperchainInput.Input memory _input) public {
        init();
        dsi.loadInput(_input);

        // Validate inputs.
        require(_input.roles.proxyAdminOwner != address(0), "zero address: proxyAdminOwner");
        require(_input.roles.protocolVersionsOwner != address(0), "zero address: protocolVersionsOwner");
        require(_input.roles.guardian != address(0), "zero address: guardian");

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
            abi.encodeCall(SuperchainConfig.initialize, (_input.roles.guardian, _input.paused))
        );

        output.protocolVersionsProxy = ProtocolVersions(address(new Proxy(address(output.superchainProxyAdmin))));
        vm.label(address(output.protocolVersionsProxy), "ProtocolVersionsProxy");
        output.superchainProxyAdmin.upgradeAndCall(
            payable(address(output.protocolVersionsProxy)),
            address(output.protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (_input.roles.protocolVersionsOwner, _input.requiredProtocolVersion, _input.recommendedProtocolVersion)
            )
        );

        // Transfer ownership of the ProxyAdmin from the deployer to the specified owner.
        output.superchainProxyAdmin.transferOwnership(_input.roles.proxyAdminOwner);

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
        dso.saveOutput(output);
    }
}
