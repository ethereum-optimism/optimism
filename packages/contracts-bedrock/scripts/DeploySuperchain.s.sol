// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { LibString } from "@solady/utils/LibString.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

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
contract DeploySuperchainInput {
    // The input struct contains all the input data required for the deployment.
    struct Input {
        Roles roles;
        bool paused;
        ProtocolVersion requiredProtocolVersion;
        ProtocolVersion recommendedProtocolVersion;
    }

    struct Roles {
        address proxyAdminOwner;
        address protocolVersionsOwner;
        address guardian;
    }

    // This flag tells us if all inputs have been set. An `input()` getter method that returns all
    // inputs reverts if this flag is false. This ensures the deploy script cannot proceed until all
    // inputs are validated and set.
    bool public inputSet = false;

    // The full input struct is kept in storage. It is not exposed because the return type would be
    // a tuple, but it's more convenient for the return type to be the struct itself. Therefore the
    // struct is exposed via the `input()` getter method.
    Input internal inputs;

    // And each field is exposed via it's own getter method. We can equivalently remove these
    // storage variables and add getter methods that return the input struct fields directly, but
    // that is more verbose with more boilerplate, especially for larger scripts with many inputs.
    // Unlike the `input()` getter, these getters do not revert if the input is not set. The caller
    // should check the `inputSet` value before calling any of these getters.
    address public proxyAdminOwner;
    address public protocolVersionsOwner;
    address public guardian;
    bool public paused;
    ProtocolVersion public requiredProtocolVersion;
    ProtocolVersion public recommendedProtocolVersion;

    // Load the input from a TOML file.
    function loadInputFile(string memory _infile) public {
        _infile;
        Input memory parsedInput;
        loadInput(parsedInput);
        require(false, "DeploySuperchainInput: loadInput is not implemented");
    }

    // Load the input from a struct.
    function loadInput(Input memory _input) public {
        // As a defensive measure, we only allow inputs to be set once.
        require(!inputSet, "DeploySuperchainInput: Input already set");

        // All assertions on inputs happen here. You cannot set any inputs in Solidity unless
        // they're all valid. For Go testing, the input and outputs
        require(_input.roles.proxyAdminOwner != address(0), "DeploySuperchainInput: Null proxyAdminOwner");
        require(_input.roles.protocolVersionsOwner != address(0), "DeploySuperchainInput: Null protocolVersionsOwner");
        require(_input.roles.guardian != address(0), "DeploySuperchainInput: Null guardian");

        // We now set all values in storage.
        inputSet = true;
        inputs = _input;

        proxyAdminOwner = _input.roles.proxyAdminOwner;
        protocolVersionsOwner = _input.roles.protocolVersionsOwner;
        guardian = _input.roles.guardian;
        paused = _input.paused;
        requiredProtocolVersion = _input.requiredProtocolVersion;
        recommendedProtocolVersion = _input.recommendedProtocolVersion;
    }

    function input() public view returns (Input memory) {
        require(inputSet, "DeploySuperchainInput: Input not set");
        return inputs;
    }
}

contract DeploySuperchainOutput {
    // The output struct contains all the output data from the deployment.
    struct Output {
        ProxyAdmin superchainProxyAdmin;
        SuperchainConfig superchainConfigImpl;
        SuperchainConfig superchainConfigProxy;
        ProtocolVersions protocolVersionsImpl;
        ProtocolVersions protocolVersionsProxy;
    }

    // We use a similar pattern as the input contract to expose outputs. Because outputs are set
    // individually, and deployment steps are modular and composable, we do not have an equivalent
    // to the overall `input` and `inputSet` variables.
    ProxyAdmin public superchainProxyAdmin;
    SuperchainConfig public superchainConfigImpl;
    SuperchainConfig public superchainConfigProxy;
    ProtocolVersions public protocolVersionsImpl;
    ProtocolVersions public protocolVersionsProxy;

    // This method lets each field be set individually. The selector of an output's getter method
    // is used to determine which field to set.
    function set(bytes4 sel, address _address) public {
        if (sel == this.superchainProxyAdmin.selector) superchainProxyAdmin = ProxyAdmin(_address);
        else if (sel == this.superchainConfigImpl.selector) superchainConfigImpl = SuperchainConfig(_address);
        else if (sel == this.superchainConfigProxy.selector) superchainConfigProxy = SuperchainConfig(_address);
        else if (sel == this.protocolVersionsImpl.selector) protocolVersionsImpl = ProtocolVersions(_address);
        else if (sel == this.protocolVersionsProxy.selector) protocolVersionsProxy = ProtocolVersions(_address);
        else revert("DeploySuperchainOutput: Unknown selector");
    }

    // Save the output to a TOML file.
    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeploySuperchainOutput: saveOutput not implemented");
    }

    function output() public view returns (Output memory) {
        return Output({
            superchainProxyAdmin: superchainProxyAdmin,
            superchainConfigImpl: superchainConfigImpl,
            superchainConfigProxy: superchainConfigProxy,
            protocolVersionsImpl: protocolVersionsImpl,
            protocolVersionsProxy: protocolVersionsProxy
        });
    }

    function checkOutput() public view {
        // Assert that all addresses are non-zero and have code.
        // We use LibString to avoid the need for adding cheatcodes to this contract.
        address[] memory addresses = new address[](5);
        addresses[0] = address(superchainProxyAdmin);
        addresses[1] = address(superchainConfigImpl);
        addresses[2] = address(superchainConfigProxy);
        addresses[3] = address(protocolVersionsImpl);
        addresses[4] = address(protocolVersionsProxy);

        for (uint256 i = 0; i < addresses.length; i++) {
            address who = addresses[i];
            require(who != address(0), string.concat("check failed: zero address at index ", LibString.toString(i)));
            require(
                who.code.length > 0, string.concat("check failed: no code at ", LibString.toHexStringChecksummed(who))
            );
        }

        // All addresses should be unique.
        for (uint256 i = 0; i < addresses.length; i++) {
            for (uint256 j = i + 1; j < addresses.length; j++) {
                string memory err =
                    string.concat("check failed: duplicates at ", LibString.toString(i), ",", LibString.toString(j));
                require(addresses[i] != addresses[j], err);
            }
        }
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
    function run(string memory _infile) public {
        // End-user without file IO, so etch the IO helper contracts.
        (DeploySuperchainInput dsi, DeploySuperchainOutput dso) = etchIOContracts();

        // Load the input file into the input contract.
        dsi.loadInputFile(_infile);

        // Run the deployment script and write outputs to the DeploySuperchainOutput contract.
        run(dsi, dso);

        // Write the output data to a file. The file
        string memory outfile = ""; // This will be derived from input file name, e.g. `foo.in.toml` -> `foo.out.toml`
        dso.writeOutputFile(outfile);
        require(false, "DeploySuperchain: run is not implemented");
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
        require(_dsi.inputSet(), "DeploySuperchain: Input not set");

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
        require(_dsi.inputSet(), "DeploySuperchain: Input not set");
        address guardian = _dsi.guardian();
        bool paused = _dsi.paused();

        ProxyAdmin superchainProxyAdmin = _dso.superchainProxyAdmin();
        SuperchainConfig superchainConfigImpl = _dso.superchainConfigImpl();
        assertValidContractAddress(address(superchainProxyAdmin));
        assertValidContractAddress(address(superchainConfigImpl));

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
        require(_dsi.inputSet(), "DeploySuperchain: Input not set");

        address protocolVersionsOwner = _dsi.protocolVersionsOwner();
        ProtocolVersion requiredProtocolVersion = _dsi.requiredProtocolVersion();
        ProtocolVersion recommendedProtocolVersion = _dsi.recommendedProtocolVersion();

        ProxyAdmin superchainProxyAdmin = _dso.superchainProxyAdmin();
        ProtocolVersions protocolVersionsImpl = _dso.protocolVersionsImpl();
        assertValidContractAddress(address(superchainProxyAdmin));
        assertValidContractAddress(address(protocolVersionsImpl));

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
        require(_dsi.inputSet(), "DeploySuperchain: Input not set");
        address proxyAdminOwner = _dsi.proxyAdminOwner();

        ProxyAdmin superchainProxyAdmin = _dso.superchainProxyAdmin();
        assertValidContractAddress(address(superchainProxyAdmin));

        vm.broadcast(msg.sender);
        superchainProxyAdmin.transferOwnership(proxyAdminOwner);
    }

    // -------- Utilities --------

    // This takes a sender and an identifier and returns a deterministic address based on the two.
    // The resulting used to etch the input and output contracts to a deterministic address based on
    // those two values, where the identifier represents the input or output contract, such as
    // `optimism.DeploySuperchainInput` or `optimism.DeployOPChainOutput`.
    function toIOAddress(address _sender, string memory _identifier) internal pure returns (address) {
        return address(uint160(uint256(keccak256(abi.encode(_sender, _identifier)))));
    }

    function etchIOContracts() internal returns (DeploySuperchainInput dsi_, DeploySuperchainOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeploySuperchainInput).runtimeCode);
        vm.etch(address(dso_), type(DeploySuperchainOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeploySuperchainInput dsi_, DeploySuperchainOutput dso_) {
        dsi_ = DeploySuperchainInput(toIOAddress(msg.sender, "optimism.DeploySuperchainInput"));
        dso_ = DeploySuperchainOutput(toIOAddress(msg.sender, "optimism.DeploySuperchainOutput"));
    }

    function assertValidContractAddress(address _address) internal view {
        require(_address != address(0), "DeploySuperchain: zero address");
        require(_address.code.length > 0, "DeploySuperchain: no code");
    }
}
