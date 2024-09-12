// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { CommonBase } from "forge-std/Base.sol";
import { stdToml } from "forge-std/StdToml.sol";

import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

// This file follows the pattern of Superchain.s.sol. Refer to that file for more details.
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
            require(_owners.length == 0, "DeployAuthSystemInput: owners already set");
            for (uint256 i = 0; i < _addrs.length; i++) {
                _owners.push(_addrs[i]);
            }
        } else {
            revert("DeployAuthSystemInput: unknown selector");
        }
    }

    function loadInputFile(string memory _infile) public {
        string memory toml = vm.readFile(_infile);

        set(this.threshold.selector, toml.readUint(".safe.threshold"));
        set(this.owners.selector, toml.readAddressArray(".safe.owners"));
    }

    function threshold() public view returns (uint256) {
        require(_threshold != 0, "DeployAuthSystemInput: threshold not set");
        return _threshold;
    }

    function owners() public view returns (address[] memory) {
        // expecting to trigger this
        require(_owners.length != 0, "DeployAuthSystemInput: owners not set");
        return _owners;
    }
}

contract DeployAuthSystemOutput is CommonBase {
    Safe internal _safe;

    function set(bytes4 sel, address _address) public {
        if (sel == this.safe.selector) _safe = Safe(payable(_address));
        else revert("DeployAuthSystemOutput: unknown selector");
    }

    function writeOutputFile(string memory _outfile) public {
        string memory out = vm.serializeAddress("outfile", "safe", address(this.safe()));
        vm.writeToml(out, _outfile);
    }

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
    function run(DeployAuthSystemInput _dasi, DeployAuthSystemOutput _daso) public {
        deploySafe(_dasi, _daso);
    }

    function deploySafe(DeployAuthSystemInput _dasi, DeployAuthSystemOutput _daso) public {
        address[] memory owners = _dasi.owners();
        uint256 threshold = _dasi.threshold();
        // Silence unused variable warnings
        owners;
        threshold;

        // TODO: replace with a real deployment. The safe deployment logic is fairly complex, so for the purposes of
        // this scaffolding PR we'll just etch the code.
        // makeAddr("safe") = 0xDC93f9959c0F9c3849461B6468B4592a19567E09
        address safe = 0xDC93f9959c0F9c3849461B6468B4592a19567E09;
        vm.label(safe, "Safe");
        vm.etch(safe, type(Safe).runtimeCode);
        vm.store(safe, bytes32(uint256(3)), bytes32(uint256(owners.length)));
        vm.store(safe, bytes32(uint256(4)), bytes32(uint256(threshold)));

        _daso.set(_daso.safe.selector, safe);
    }

    // This etches the IO contracts into memory so that we can use them in tests. When using file IO
    // we don't need to call this directly, as the `DeployAuthSystem.run(file, file)` entrypoint
    // handles it. But when interacting with the script programmatically (e.g. in a Solidity test),
    // this must be called.
    function etchIOContracts() public returns (DeployAuthSystemInput dasi_, DeployAuthSystemOutput daso_) {
        (dasi_, daso_) = getIOContracts();
        vm.etch(address(dasi_), type(DeployAuthSystemInput).runtimeCode);
        vm.etch(address(daso_), type(DeployAuthSystemOutput).runtimeCode);
        vm.allowCheatcodes(address(dasi_));
        vm.allowCheatcodes(address(daso_));
    }

    // This returns the addresses of the IO contracts for this script.
    function getIOContracts() public view returns (DeployAuthSystemInput dasi_, DeployAuthSystemOutput daso_) {
        dasi_ = DeployAuthSystemInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployAuthSystemInput"));
        daso_ = DeployAuthSystemOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployAuthSystemOutput"));
    }
}
