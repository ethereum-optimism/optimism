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
