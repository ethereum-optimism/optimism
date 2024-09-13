// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { CommonBase } from "forge-std/Base.sol";
import { stdToml } from "forge-std/StdToml.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

// This comment block defines the requirements and rationale for the architecture used in this forge
// script, along with other scripts that are being written as new Superchain-first deploy scripts to
// complement the OP Stack Manager. The script architecture is a bit different than a standard forge
// deployment script.
//
// There are three categories of users that are expected to interact with the scripts:
//   1. End users that want to run live contract deployments.
//   2. Solidity developers that want to use or test these scripts in a standard forge test environment.
//   3. Go developers that want to run the deploy scripts as part of e2e testing with other aspects of the OP Stack.
//
// We want each user to interact with the scripts in the way that's simplest for their use case:
//   1. End users: TOML input files that define config, and TOML output files with all output data.
//   2. Solidity developers: Direct calls to the script, with the input and output contracts configured.
//   3. Go developers: The forge scripts can be executed directly in Go.
//
// The following architecture is used to meet the requirements of each user. We use this file's
// `DeploySuperchain` script as an example, but it applies to other scripts as well.
//
// This `DeploySuperchain.s.sol` file contains three contracts:
//   1. `DeploySuperchainInput`: Responsible for parsing, storing, and exposing the input data.
//   2. `DeploySuperchainOutput`: Responsible for storing and exposing the output data.
//   3. `DeploySuperchain`: The core script that executes the deployment. It reads inputs from the
//      input contract, and writes outputs to the output contract.
//
// Because the core script performs calls to the input and output contracts, Go developers can
// intercept calls to these addresses (analogous to how forge intercepts calls to the `Vm` address
// to execute cheatcodes), to avoid the need for file I/O or hardcoding the input/output values.
//
// Public getter methods on the input and output contracts allow individual fields to be accessed
// in a strong, type-safe manner (as opposed to a single struct getter where the caller may
// inadvertently transpose two addresses, for example).
//
// Each deployment step in the core deploy script is modularized into its own function that performs
// the deploy and sets the output on the Output contract, allowing for easy composition and testing
// of deployment steps. The output setter methods requires keying off the four-byte selector of
// each output field's getter method, ensuring that the output is set for the correct field and
// minimizing the amount of boilerplate needed for each output field.
//
// This script doubles as a reference for documenting the pattern used and therefore contains
// comments explaining the patterns used. Other scripts are not expected to have this level of
// documentation.
//
// Additionally, we intentionally use "Input" and "Output" terminology to clearly distinguish these
// scripts from the existing ones that use the "Config" and "Artifacts" terminology. Within scripts
// we use variable names that are shorthand for the full contract names, for example:
//   - `dsi` for DeploySuperchainInput
//   - `dso` for DeploySuperchainOutput
//   - `dio` for DeployImplementationsInput
//   - `dio` for DeployImplementationsOutput
//   - `doo` for DeployOPChainInput
//   - `doo` for DeployOPChainOutput
//   - etc.
contract DeploySuperchainInput is CommonBase {
    using stdToml for string;

    // All inputs are set in storage individually. We put any roles first, followed by the remaining
    // inputs. Inputs are internal and prefixed with an underscore, because we will expose a getter
    // method that returns the input value. We use a getter method to allow us to make assertions on
    // the input to ensure it's valid before returning it. We also intentionally do not use a struct
    // to hold all inputs, because as features are developed the set of inputs will change, and
    // modifying structs in Solidity is not very simple.

    // Role inputs.
    address internal _guardian;
    address internal _protocolVersionsOwner;
    address internal _proxyAdminOwner;

    // Other inputs.
    bool internal _paused;
    ProtocolVersion internal _recommendedProtocolVersion;
    ProtocolVersion internal _requiredProtocolVersion;

    // These `set` methods let each input be set individually. The selector of an input's getter method
    // is used to determine which field to set.
    function set(bytes4 _sel, address _address) public {
        require(_address != address(0), "DeploySuperchainInput: cannot set zero address");
        if (_sel == this.guardian.selector) _guardian = _address;
        else if (_sel == this.protocolVersionsOwner.selector) _protocolVersionsOwner = _address;
        else if (_sel == this.proxyAdminOwner.selector) _proxyAdminOwner = _address;
        else revert("DeploySuperchainInput: unknown selector");
    }

    function set(bytes4 _sel, bool _value) public {
        if (_sel == this.paused.selector) _paused = _value;
        else revert("DeploySuperchainInput: unknown selector");
    }

    function set(bytes4 _sel, ProtocolVersion _value) public {
        require(ProtocolVersion.unwrap(_value) != 0, "DeploySuperchainInput: cannot set null protocol version");
        if (_sel == this.recommendedProtocolVersion.selector) _recommendedProtocolVersion = _value;
        else if (_sel == this.requiredProtocolVersion.selector) _requiredProtocolVersion = _value;
        else revert("DeploySuperchainInput: unknown selector");
    }

    // Load the input from a TOML file.
    // When setting inputs from a TOML file, we use the setter methods instead of writing directly
    // to storage. This allows us to validate each input as it is set.
    function loadInputFile(string memory _infile) public {
        string memory toml = vm.readFile(_infile);

        // Parse and set role inputs.
        set(this.guardian.selector, toml.readAddress(".roles.guardian"));
        set(this.protocolVersionsOwner.selector, toml.readAddress(".roles.protocolVersionsOwner"));
        set(this.proxyAdminOwner.selector, toml.readAddress(".roles.proxyAdminOwner"));

        // Parse and set other inputs.
        set(this.paused.selector, toml.readBool(".paused"));

        uint256 recVersion = toml.readUint(".recommendedProtocolVersion");
        set(this.recommendedProtocolVersion.selector, ProtocolVersion.wrap(recVersion));

        uint256 reqVersion = toml.readUint(".requiredProtocolVersion");
        set(this.requiredProtocolVersion.selector, ProtocolVersion.wrap(reqVersion));
    }

    // Each input field is exposed via it's own getter method. Using public storage variables here
    // would be less verbose, but would also be more error-prone, as it would require the caller to
    // validate that each input is set before accessing it. With getter methods, we can automatically
    // validate that each input is set before allowing any field to be accessed.

    function proxyAdminOwner() public view returns (address) {
        require(_proxyAdminOwner != address(0), "DeploySuperchainInput: proxyAdminOwner not set");
        return _proxyAdminOwner;
    }

    function protocolVersionsOwner() public view returns (address) {
        require(_protocolVersionsOwner != address(0), "DeploySuperchainInput: protocolVersionsOwner not set");
        return _protocolVersionsOwner;
    }

    function guardian() public view returns (address) {
        require(_guardian != address(0), "DeploySuperchainInput: guardian not set");
        return _guardian;
    }

    function paused() public view returns (bool) {
        return _paused;
    }

    function requiredProtocolVersion() public view returns (ProtocolVersion) {
        require(
            ProtocolVersion.unwrap(_requiredProtocolVersion) != 0,
            "DeploySuperchainInput: requiredProtocolVersion not set"
        );
        return _requiredProtocolVersion;
    }

    function recommendedProtocolVersion() public view returns (ProtocolVersion) {
        require(
            ProtocolVersion.unwrap(_recommendedProtocolVersion) != 0,
            "DeploySuperchainInput: recommendedProtocolVersion not set"
        );
        return _recommendedProtocolVersion;
    }
}

contract DeploySuperchainOutput is CommonBase {
    // All outputs are stored in storage individually, with the same rationale as doing so for
    // inputs, and the same pattern is used below to expose the outputs.
    ProtocolVersions internal _protocolVersionsImpl;
    ProtocolVersions internal _protocolVersionsProxy;
    SuperchainConfig internal _superchainConfigImpl;
    SuperchainConfig internal _superchainConfigProxy;
    ProxyAdmin internal _superchainProxyAdmin;

    // This method lets each field be set individually. The selector of an output's getter method
    // is used to determine which field to set.
    function set(bytes4 sel, address _address) public {
        require(_address != address(0), "DeploySuperchainOutput: cannot set zero address");
        if (sel == this.superchainProxyAdmin.selector) _superchainProxyAdmin = ProxyAdmin(_address);
        else if (sel == this.superchainConfigImpl.selector) _superchainConfigImpl = SuperchainConfig(_address);
        else if (sel == this.superchainConfigProxy.selector) _superchainConfigProxy = SuperchainConfig(_address);
        else if (sel == this.protocolVersionsImpl.selector) _protocolVersionsImpl = ProtocolVersions(_address);
        else if (sel == this.protocolVersionsProxy.selector) _protocolVersionsProxy = ProtocolVersions(_address);
        else revert("DeploySuperchainOutput: unknown selector");
    }

    // Save the output to a TOML file.
    // We fetch the output values using external calls to the getters to verify that all outputs are
    // set correctly before writing them to the file.
    function writeOutputFile(string memory _outfile) public {
        string memory key = "dso-outfile";
        vm.serializeAddress(key, "superchainProxyAdmin", address(this.superchainProxyAdmin()));
        vm.serializeAddress(key, "superchainConfigImpl", address(this.superchainConfigImpl()));
        vm.serializeAddress(key, "superchainConfigProxy", address(this.superchainConfigProxy()));
        vm.serializeAddress(key, "protocolVersionsImpl", address(this.protocolVersionsImpl()));
        string memory out = vm.serializeAddress(key, "protocolVersionsProxy", address(this.protocolVersionsProxy()));
        vm.writeToml(out, _outfile);
    }

    // This function can be called to ensure all outputs are correct. Similar to `writeOutputFile`,
    // it fetches the output values using external calls to the getter methods for safety.
    function checkOutput() public {
        address[] memory addrs = Solarray.addresses(
            address(this.superchainProxyAdmin()),
            address(this.superchainConfigImpl()),
            address(this.superchainConfigProxy()),
            address(this.protocolVersionsImpl()),
            address(this.protocolVersionsProxy())
        );
        DeployUtils.assertValidContractAddresses(addrs);

        // To read the implementations we prank as the zero address due to the proxyCallIfNotAdmin modifier.
        vm.startPrank(address(0));
        address actualSuperchainConfigImpl = Proxy(payable(address(_superchainConfigProxy))).implementation();
        address actualProtocolVersionsImpl = Proxy(payable(address(_protocolVersionsProxy))).implementation();
        vm.stopPrank();

        require(actualSuperchainConfigImpl == address(_superchainConfigImpl), "100");
        require(actualProtocolVersionsImpl == address(_protocolVersionsImpl), "200");
    }

    function superchainProxyAdmin() public view returns (ProxyAdmin) {
        // This does not have to be a contract address, it could be an EOA.
        return _superchainProxyAdmin;
    }

    function superchainConfigImpl() public view returns (SuperchainConfig) {
        DeployUtils.assertValidContractAddress(address(_superchainConfigImpl));
        return _superchainConfigImpl;
    }

    function superchainConfigProxy() public view returns (SuperchainConfig) {
        DeployUtils.assertValidContractAddress(address(_superchainConfigProxy));
        return _superchainConfigProxy;
    }

    function protocolVersionsImpl() public view returns (ProtocolVersions) {
        DeployUtils.assertValidContractAddress(address(_protocolVersionsImpl));
        return _protocolVersionsImpl;
    }

    function protocolVersionsProxy() public view returns (ProtocolVersions) {
        DeployUtils.assertValidContractAddress(address(_protocolVersionsProxy));
        return _protocolVersionsProxy;
    }
}

// For all broadcasts in this script we explicitly specify the deployer as `msg.sender` because for
// testing we deploy this script from a test contract. If we provide no argument, the foundry
// default sender would be the broadcaster during test, but the broadcaster needs to be the deployer
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

    // This entrypoint is useful for testing purposes, as it doesn't use any file I/O.
    function run(DeploySuperchainInput _dsi, DeploySuperchainOutput _dso) public {
        // Notice that we do not do any explicit verification here that inputs are set. This is because
        // the verification happens elsewhere:
        //   - Getter methods on the input contract provide sanity checks that values are set, when applicable.
        //   - The individual methods below that we use to compose the deployment are responsible for handling
        //     their own verification.
        // This pattern ensures that other deploy scripts that might compose these contracts and
        // methods in different ways are still protected from invalid inputs without need to implement
        // additional verification logic.

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
        // contract. If we provide no argument, the foundry default sender would be the broadcaster during test, but the
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

    // This etches the IO contracts into memory so that we can use them in tests. When using file IO
    // we don't need to call this directly, as the `DeploySuperchain.run(file, file)` entrypoint
    // handles it. But when interacting with the script programmatically (e.g. in a Solidity test),
    // this must be called.
    function etchIOContracts() public returns (DeploySuperchainInput dsi_, DeploySuperchainOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeploySuperchainInput).runtimeCode);
        vm.etch(address(dso_), type(DeploySuperchainOutput).runtimeCode);
        vm.allowCheatcodes(address(dsi_));
        vm.allowCheatcodes(address(dso_));
    }

    // This returns the addresses of the IO contracts for this script.
    function getIOContracts() public view returns (DeploySuperchainInput dsi_, DeploySuperchainOutput dso_) {
        dsi_ = DeploySuperchainInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeploySuperchainInput"));
        dso_ = DeploySuperchainOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeploySuperchainOutput"));
    }
}
