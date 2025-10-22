// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { Safe } from "safe-contracts/Safe.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

contract DeployAuthSystem2 is Script {
    struct Input {
        uint256 threshold;
        address[] owners;
    }

    struct Output {
        Safe safe;
    }

    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deploySafe(_input, output_);

        assertValidOutput(_input, output_);
    }

    function assertValidInput(Input memory _input) internal pure {
        require(_input.owners.length != 0, "DeployAuthSystem: owners not set");
        require(_input.threshold != 0, "DeployAuthSystem: threshold not set");
        require(_input.threshold <= _input.owners.length, "DeployAuthSystem: threshold too large");

        for (uint256 i = 0; i < _input.owners.length; i++) {
            require(_input.owners[i] != address(0), "DeployAuthSystem: owner not set");
        }

        DeployUtils.assertUniqueAddresses(_input.owners);
    }

    function assertValidOutput(Input memory, Output memory _output) internal view {
        DeployUtils.assertValidContractAddress(address(_output.safe));
    }

    function deploySafe(Input memory _input, Output memory _output) public {
        //
        // TODO: replace with a real deployment.
        //
        address safe = makeAddr("safe");
        vm.etch(safe, type(Safe).runtimeCode);
        vm.store(safe, bytes32(uint256(3)), bytes32(_input.owners.length));
        vm.store(safe, bytes32(uint256(4)), bytes32(_input.threshold));

        _output.safe = Safe(payable(safe));
    }
}
