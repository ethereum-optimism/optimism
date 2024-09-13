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
