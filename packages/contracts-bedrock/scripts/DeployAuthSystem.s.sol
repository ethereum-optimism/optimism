// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { CommonBase } from "forge-std/Base.sol";
import { stdToml } from "forge-std/StdToml.sol";

import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

contract DeployAuthSystemInput is CommonBase {
    using stdToml for string;

    // Generic safe inputs
    // Note: these will need to be replaced with settings specific to the different Safes in the system.
    uint256 internal _threshold;
    address[] internal _owners;

    function set(bytes4 _sel, uint256 _value) public {
        if (_sel == this.threshold.selector) _threshold = _value;
        else revert("DeployAuthSystemInput: unknown selector");
    }

    function set(bytes4 _sel, address[] memory _addrs) public {
        if (_sel == this.owners.selector) {
            for (uint256 i = 0; i < _addrs.length; i++) {
                _owners.push(_addrs[i]);
            }
        } else {
            revert("DeployAuthSystemInput: unknown selector");
        }
    }

    // Load the input from a TOML file.
    // When setting inputs from a TOML file, we use the setter methods instead of writing directly
    // to storage. This allows us to validate each input as it is set.
    function loadInputFile(string memory _infile) public {
        string memory toml = vm.readFile(_infile);

        // Parse and set role inputs.
        set(this.threshold.selector, toml.readUint(".safe.threshold"));
        set(this.owners.selector, toml.readAddressArray(".safe.owners"));
    }

    function threshold() public view returns (uint256) {
        require(_threshold != 0, "DeployAuthSystemInput: threshold not set");
        return _threshold;
    }

    function owners() public view returns (address[] memory) {
        require(_owners.length != 0, "DeployAuthSystemInput: owners not set");
        return _owners;
    }
}

contract DeployAuthSystemOutput is CommonBase {
    Safe internal _safe;

    // This method lets each field be set individually. The selector of an output's getter method
    // is used to determine which field to set.
    function set(bytes4 sel, address _address) public {
        if (sel == this.safe.selector) _safe = Safe(payable(_address));
        else revert("DeployAuthSystemOutput: unknown selector");
    }

    // Save the output to a TOML file.
    // We fetch the output values using external calls to the getters to verify that all outputs are
    // set correctly before writing them to the file.
    function writeOutputFile(string memory _outfile) public {
        string memory out = vm.serializeAddress("outfile", "safe", address(this.safe()));
        vm.writeToml(out, _outfile);
    }

    // This function can be called to ensure all outputs are correct. Similar to `writeOutputFile`,
    // it fetches the output values using external calls to the getter methods for safety.
    function checkOutput() public view {
        address[] memory addrs = Solarray.addresses(address(this.safe()));
        DeployUtils.assertValidContractAddresses(addrs);
    }

    function safe() public view returns (Safe) {
        DeployUtils.assertValidContractAddress(address(_safe));
        return _safe;
    }
}

// For all broadcasts in this script we explicitly specify the deployer as `msg.sender` because for
// testing we deploy this script from a test contract. If we provide no argument, the foundry
// default sender would be the broadcaster during test, but the broadcaster needs to be the deployer
// since they are set to the initial proxy admin owner.
contract DeployAuthSystem is Script {
    // -------- Core Deployment Methods --------

    // This entrypoint is for end-users to deploy from an input file and write to an output file.
    // In this usage, we don't need the input and output contract functionality, so we deploy them
    // here and abstract that architectural detail away from the end user.
    function run(string memory _infile, string memory _outfile) public {
        // End-user without file IO, so etch the IO helper contracts.
        (DeployAuthSystemInput dasi, DeployAuthSystemOutput daso) = etchIOContracts();

        // Load the input file into the input contract.
        dasi.loadInputFile(_infile);

        // Run the deployment script and write outputs to the DeployAuthSystemOutput contract.
        run(dasi, daso);

        // Write the output data to a file.
        daso.writeOutputFile(_outfile);
    }

    // This entrypoint is useful for testing purposes, as it doesn't use any file I/O.
    function run(DeployAuthSystemInput, DeployAuthSystemOutput _daso) public {
        // Deploy the Safe contract
        // TODO: replace with a real deployment. The safe deployment logic is fairly complex, so for the purposes of
        // this scaffolding PR we'll just etch the code.
        // makeAddr("safe") = 0xDC93f9959c0F9c3849461B6468B4592a19567E09
        vm.etch(address(0xDC93f9959c0F9c3849461B6468B4592a19567E09), type(Safe).runtimeCode);
        _daso.set(_daso.safe.selector, address(0xDC93f9959c0F9c3849461B6468B4592a19567E09));
    }

    // -------- Utilities --------

    // This etches the IO contracts into memory so that we can use them in tests. When using file IO
    // we don't need to call this directly, as the `DeployAuthSystem.run(file, file)` entrypoint
    // handles it. But when interacting with the script programmatically (e.g. in a Solidity test),
    // this must be called.
    function etchIOContracts() public returns (DeployAuthSystemInput dsi_, DeployAuthSystemOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeployAuthSystemInput).runtimeCode);
        vm.etch(address(dso_), type(DeployAuthSystemOutput).runtimeCode);
        vm.allowCheatcodes(address(dsi_));
        vm.allowCheatcodes(address(dso_));
    }

    // This returns the addresses of the IO contracts for this script.
    function getIOContracts() public view returns (DeployAuthSystemInput dsi_, DeployAuthSystemOutput dso_) {
        dsi_ = DeployAuthSystemInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployAuthSystemInput"));
        dso_ = DeployAuthSystemOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployAuthSystemOutput"));
    }
}
