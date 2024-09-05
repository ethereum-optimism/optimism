// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { CommonBase } from "forge-std/Base.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";
/**
 * This comment block defines the requirements and rationale for the architecture used in this forge
 * script, along with other scripts that are being written as new Superchain-first deploy scripts to
 * complement the OP Stack Manager. The script architecture is a bit different than a standard forge
 * deployment script.
 *
 * There are three categories of users that are expected to interact with the scripts:
 *   1. End users that want to run live contract deployments.
 *   2. Solidity developers that want to use or test these script in a standard forge test environment.
 *   3. Go developers that want to run the deploy scripts as part of e2e testing with other aspects of the OP Stack.
 *
 * We want each user to interact with the scripts in the way that's simplest for their use case:
 *   1. End users: TOML input files that define config, and TOML output files with all output data.
 *   2. Solidity developers: Direct calls to the script with input structs and output structs.
 *   3. Go developers: The forge scripts can be executed directly in Go.
 *
 * The following architecture is used to meet the requirements of each user. We use this file's
 * `DeploySuperchain` script as an example, but it applies to other scripts as well.
 *
 * This `DeploySuperchain.s.sol` file contains three contracts:
 *   1. `DeploySuperchainInput`: Responsible for parsing, storing, and exposing the input data.
 *   2. `DeploySuperchainOutput`: Responsible for storing and exposing the output data.
 *   3. `DeploySuperchain`: The core script that executes the deployment. It reads inputs from the
 *      input contract, and writes outputs to the output contract.
 *
 * Because the core script performs calls to the input and output contracts, Go developers can
 * intercept calls to these addresses (analogous to how forge intercepts calls to the `Vm` address
 * to execute cheatcodes), to avoid the need for file I/O or hardcoding the input/output structs.
 *
 * Public getter methods on the input and output contracts allow individual fields to be accessed
 * in a strong, type-safe manner (as opposed to a single struct getter where the caller may
 * inadvertently transpose two addresses, for example).
 *
 * Each deployment step in the core deploy script is modularized into its own function that performs
 * the deploy and sets the output on the Output contract, allowing for easy composition and testing
 * of deployment steps. The output setter methods requires keying off the four-byte selector of the
 * each output field's getter method, ensuring that the output is set for the correct field and
 * minimizing the amount of boilerplate needed for each output field.
 *
 * This script doubles as a reference for documenting the pattern used and therefore contains
 * comments explaining the patterns used. Other scripts are not expected to have this level of
 * documentation.
 *
 * Additionally, we intentionally use "Input" and "Output" terminology to clearly distinguish these
 * scripts from the existing ones that "Config" and "Artifacts" terminology.
 */

contract DeploySuperchainInput is CommonBase {
    // The input struct contains all the input data required for the deployment.
    // The fields must be in alphabetical order for vm.parseToml to work.
    struct Input {
        bool paused;
        ProtocolVersion recommendedProtocolVersion;
        ProtocolVersion requiredProtocolVersion;
        Roles roles;
    }

    struct Roles {
        address guardian;
        address protocolVersionsOwner;
        address proxyAdminOwner;
    }

    // This flag tells us if all inputs have been set. An `input()` getter method that returns all
    // inputs reverts if this flag is false. This ensures the deploy script cannot proceed until all
    // inputs are validated and set.
    bool public inputSet = false;

    // The full input struct is kept in storage. It is not exposed because the return type would be
    // a tuple, and it's more convenient for the return type to be the struct itself. Therefore the
    // struct is exposed via the `input()` getter method below.
    Input internal inputs;

    // Load the input from a TOML file.
    function loadInputFile(string memory _infile) public {
        string memory toml = vm.readFile(_infile);
        bytes memory data = vm.parseToml(toml);
        Input memory parsedInput = abi.decode(data, (Input));
        loadInput(parsedInput);
    }

    // Load the input from a struct.
    function loadInput(Input memory _input) public {
        // As a defensive measure, we only allow inputs to be set once.
        require(!inputSet, "DeploySuperchainInput: input already set");

        // All assertions on inputs happen here. You cannot set any inputs in Solidity unless
        // they're all valid. For Go testing, the input and outputs are set individually by
        // treating the input and output contracts as precompiles and intercepting calls to them.
        require(_input.roles.proxyAdminOwner != address(0), "DeploySuperchainInput: null proxyAdminOwner");
        require(_input.roles.protocolVersionsOwner != address(0), "DeploySuperchainInput: null protocolVersionsOwner");
        require(_input.roles.guardian != address(0), "DeploySuperchainInput: null guardian");

        // We now set all values in storage.
        inputSet = true;
        inputs = _input;
    }

    function assertInputSet() internal view {
        require(inputSet, "DeploySuperchainInput: input not set");
    }

    // This exposes the full input data as a struct, and it reverts if the input has not been set.
    function input() public view returns (Input memory) {
        assertInputSet();
        return inputs;
    }

    // Each field of the input struct is exposed via it's own getter method. Using public storage
    // variables here would be more verbose, but would also be more error-prone, as it would
    // require the caller to remember to check the `inputSet` flag before accessing any of the
    // fields. With getter methods, we can be sure that the input is set before accessing any field.

    function proxyAdminOwner() public view returns (address) {
        assertInputSet();
        return inputs.roles.proxyAdminOwner;
    }

    function protocolVersionsOwner() public view returns (address) {
        assertInputSet();
        return inputs.roles.protocolVersionsOwner;
    }

    function guardian() public view returns (address) {
        assertInputSet();
        return inputs.roles.guardian;
    }

    function paused() public view returns (bool) {
        assertInputSet();
        return inputs.paused;
    }

    function requiredProtocolVersion() public view returns (ProtocolVersion) {
        assertInputSet();
        return inputs.requiredProtocolVersion;
    }

    function recommendedProtocolVersion() public view returns (ProtocolVersion) {
        assertInputSet();
        return inputs.recommendedProtocolVersion;
    }
}

contract DeploySuperchainOutput is CommonBase {
    // The output struct contains all the output data from the deployment.
    // The fields must be in alphabetical order for vm.parseToml to work.
    struct Output {
        ProtocolVersions protocolVersionsImpl;
        ProtocolVersions protocolVersionsProxy;
        SuperchainConfig superchainConfigImpl;
        SuperchainConfig superchainConfigProxy;
        ProxyAdmin superchainProxyAdmin;
    }

    // We use a similar pattern as the input contract to expose outputs. Because outputs are set
    // individually, and deployment steps are modular and composable, we do not have an equivalent
    // to the overall `inputSet` variable. However, we do hold everything in a struct, then
    // similarly expose each field via a getter method. This getter method reverts if the output has
    // not been set, ensuring that the caller cannot access any output fields until they have been set.
    Output internal outputs;

    // This method lets each field be set individually. The selector of an output's getter method
    // is used to determine which field to set.
    function set(bytes4 sel, address _address) public {
        if (sel == this.superchainProxyAdmin.selector) outputs.superchainProxyAdmin = ProxyAdmin(_address);
        else if (sel == this.superchainConfigImpl.selector) outputs.superchainConfigImpl = SuperchainConfig(_address);
        else if (sel == this.superchainConfigProxy.selector) outputs.superchainConfigProxy = SuperchainConfig(_address);
        else if (sel == this.protocolVersionsImpl.selector) outputs.protocolVersionsImpl = ProtocolVersions(_address);
        else if (sel == this.protocolVersionsProxy.selector) outputs.protocolVersionsProxy = ProtocolVersions(_address);
        else revert("DeploySuperchainOutput: unknown selector");
    }

    // Save the output to a TOML file.
    function writeOutputFile(string memory _outfile) public {
        string memory key = "dso-outfile";
        vm.serializeAddress(key, "superchainProxyAdmin", address(outputs.superchainProxyAdmin));
        vm.serializeAddress(key, "superchainConfigImpl", address(outputs.superchainConfigImpl));
        vm.serializeAddress(key, "superchainConfigProxy", address(outputs.superchainConfigProxy));
        vm.serializeAddress(key, "protocolVersionsImpl", address(outputs.protocolVersionsImpl));
        string memory out = vm.serializeAddress(key, "protocolVersionsProxy", address(outputs.protocolVersionsProxy));
        vm.writeToml(out, _outfile);
    }

    function output() public view returns (Output memory) {
        return outputs;
    }

    function checkOutput() public view {
        address[] memory addrs = Solarray.addresses(
            address(outputs.superchainProxyAdmin),
            address(outputs.superchainConfigImpl),
            address(outputs.superchainConfigProxy),
            address(outputs.protocolVersionsImpl),
            address(outputs.protocolVersionsProxy)
        );
        DeployUtils.assertValidContractAddresses(addrs);
    }

    function superchainProxyAdmin() public view returns (ProxyAdmin) {
        DeployUtils.assertValidContractAddress(address(outputs.superchainProxyAdmin));
        return outputs.superchainProxyAdmin;
    }

    function superchainConfigImpl() public view returns (SuperchainConfig) {
        DeployUtils.assertValidContractAddress(address(outputs.superchainConfigImpl));
        return outputs.superchainConfigImpl;
    }

    function superchainConfigProxy() public view returns (SuperchainConfig) {
        DeployUtils.assertValidContractAddress(address(outputs.superchainConfigProxy));
        return outputs.superchainConfigProxy;
    }

    function protocolVersionsImpl() public view returns (ProtocolVersions) {
        DeployUtils.assertValidContractAddress(address(outputs.protocolVersionsImpl));
        return outputs.protocolVersionsImpl;
    }

    function protocolVersionsProxy() public view returns (ProtocolVersions) {
        DeployUtils.assertValidContractAddress(address(outputs.protocolVersionsProxy));
        return outputs.protocolVersionsProxy;
    }
}

// For all broadcasts in this script we explicitly specify the deployer as `msg.sender` because for
// testing we deploy this script from a test contract. If we provide no argument, the foundry
// default sender is be the broadcaster during test, but the broadcaster needs to be the deployer
// since they are set to the initial proxy admin owner.
contract DeploySuperchain is Script {
    // -------- Core Deployment Methods --------

    // This entrypoint is for end-users to deploy from an input file and write to an output file.
    // In this usage, we don't need the input and output contract functionality, so we deploy them
    // here and abstract that architectural detail away from the end user.
    function run(string memory _infile, string memory _outfile) public {
        // End-user without file IO, so etch the IO helper contracts.
        (DeploySuperchainInput dsi, DeploySuperchainOutput dso) = etchIOContracts();

        // Load the input file into the input contract.
        dsi.loadInputFile(_infile);

        // Run the deployment script and write outputs to the DeploySuperchainOutput contract.
        run(dsi, dso);

        // Write the output data to a file.
        dso.writeOutputFile(_outfile);
    }

    // This entrypoint is for use with Solidity tests, where the input and outputs are structs.
    function run(DeploySuperchainInput.Input memory _input) public returns (DeploySuperchainOutput.Output memory) {
        // Solidity without file IO, so etch the IO helper contracts.
        (DeploySuperchainInput dsi, DeploySuperchainOutput dso) = etchIOContracts();

        // Load the input struct into the input contract.
        dsi.loadInput(_input);

        // Run the deployment script and write outputs to the DeploySuperchainOutput contract.
        run(dsi, dso);

        // Return the output struct from the output contract.
        return dso.output();
    }

    // This entrypoint is useful for testing purposes, as it doesn't use any file I/O.
    function run(DeploySuperchainInput _dsi, DeploySuperchainOutput _dso) public {
        // Verify that the input contract has been set.
        require(_dsi.inputSet(), "DeploySuperchain: input not set");

        // Deploy the proxy admin, with the owner set to the deployer.
        deploySuperchainProxyAdmin(_dsi, _dso);

        // Deploy and initialize the superchain contracts.
        deploySuperchainImplementationContracts(_dsi, _dso);
        deployAndInitializeSuperchainConfig(_dsi, _dso);
        deployAndInitializeProtocolVersions(_dsi, _dso);

        // Transfer ownership of the ProxyAdmin from the deployer to the specified owner.
        transferProxyAdminOwnership(_dsi, _dso);

        // Output assertions, to make sure outputs were assigned correctly.
        _dso.checkOutput();
    }

    // -------- Deployment Steps --------

    function deploySuperchainProxyAdmin(DeploySuperchainInput, DeploySuperchainOutput _dso) public {
        // Deploy the proxy admin, with the owner set to the deployer.
        // We explicitly specify the deployer as `msg.sender` because for testing we deploy this script from a test
        // contract. If we provide no argument, the foundry default sender is be the broadcaster during test, but the
        // broadcaster needs to be the deployer since they are set to the initial proxy admin owner.
        vm.broadcast(msg.sender);
        ProxyAdmin superchainProxyAdmin = new ProxyAdmin(msg.sender);

        vm.label(address(superchainProxyAdmin), "SuperchainProxyAdmin");
        _dso.set(_dso.superchainProxyAdmin.selector, address(superchainProxyAdmin));
    }

    function deploySuperchainImplementationContracts(DeploySuperchainInput, DeploySuperchainOutput _dso) public {
        // Deploy implementation contracts.
        vm.startBroadcast(msg.sender);
        SuperchainConfig superchainConfigImpl = new SuperchainConfig();
        ProtocolVersions protocolVersionsImpl = new ProtocolVersions();
        vm.stopBroadcast();

        vm.label(address(superchainConfigImpl), "SuperchainConfigImpl");
        vm.label(address(protocolVersionsImpl), "ProtocolVersionsImpl");

        _dso.set(_dso.superchainConfigImpl.selector, address(superchainConfigImpl));
        _dso.set(_dso.protocolVersionsImpl.selector, address(protocolVersionsImpl));
    }

    function deployAndInitializeSuperchainConfig(DeploySuperchainInput _dsi, DeploySuperchainOutput _dso) public {
        address guardian = _dsi.guardian();
        bool paused = _dsi.paused();

        ProxyAdmin superchainProxyAdmin = _dso.superchainProxyAdmin();
        SuperchainConfig superchainConfigImpl = _dso.superchainConfigImpl();

        vm.startBroadcast(msg.sender);
        SuperchainConfig superchainConfigProxy = SuperchainConfig(address(new Proxy(address(superchainProxyAdmin))));
        superchainProxyAdmin.upgradeAndCall(
            payable(address(superchainConfigProxy)),
            address(superchainConfigImpl),
            abi.encodeCall(SuperchainConfig.initialize, (guardian, paused))
        );
        vm.stopBroadcast();

        vm.label(address(superchainConfigProxy), "SuperchainConfigProxy");
        _dso.set(_dso.superchainConfigProxy.selector, address(superchainConfigProxy));
    }

    function deployAndInitializeProtocolVersions(DeploySuperchainInput _dsi, DeploySuperchainOutput _dso) public {
        address protocolVersionsOwner = _dsi.protocolVersionsOwner();
        ProtocolVersion requiredProtocolVersion = _dsi.requiredProtocolVersion();
        ProtocolVersion recommendedProtocolVersion = _dsi.recommendedProtocolVersion();

        ProxyAdmin superchainProxyAdmin = _dso.superchainProxyAdmin();
        ProtocolVersions protocolVersionsImpl = _dso.protocolVersionsImpl();

        vm.startBroadcast(msg.sender);
        ProtocolVersions protocolVersionsProxy = ProtocolVersions(address(new Proxy(address(superchainProxyAdmin))));
        superchainProxyAdmin.upgradeAndCall(
            payable(address(protocolVersionsProxy)),
            address(protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (protocolVersionsOwner, requiredProtocolVersion, recommendedProtocolVersion)
            )
        );
        vm.stopBroadcast();

        vm.label(address(protocolVersionsProxy), "ProtocolVersionsProxy");
        _dso.set(_dso.protocolVersionsProxy.selector, address(protocolVersionsProxy));
    }

    function transferProxyAdminOwnership(DeploySuperchainInput _dsi, DeploySuperchainOutput _dso) public {
        address proxyAdminOwner = _dsi.proxyAdminOwner();

        ProxyAdmin superchainProxyAdmin = _dso.superchainProxyAdmin();
        DeployUtils.assertValidContractAddress(address(superchainProxyAdmin));

        vm.broadcast(msg.sender);
        superchainProxyAdmin.transferOwnership(proxyAdminOwner);
    }

    // -------- Utilities --------

    function etchIOContracts() internal returns (DeploySuperchainInput dsi_, DeploySuperchainOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeploySuperchainInput).runtimeCode);
        vm.etch(address(dso_), type(DeploySuperchainOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeploySuperchainInput dsi_, DeploySuperchainOutput dso_) {
        dsi_ = DeploySuperchainInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeploySuperchainInput"));
        dso_ = DeploySuperchainOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeploySuperchainOutput"));
    }
}
