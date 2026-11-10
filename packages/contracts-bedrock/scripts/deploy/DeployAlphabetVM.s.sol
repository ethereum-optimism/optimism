// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { Claim } from "src/dispute/lib/Types.sol";

contract DeployAlphabetVM is Script {
    struct Input {
        bytes32 absolutePrestate;
        IPreimageOracle preimageOracle;
    }

    struct Output {
        AlphabetVM alphabetVM;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        Claim absolutePrestate = Claim.wrap(_input.absolutePrestate);
        IPreimageOracle preimageOracle = _input.preimageOracle;

        // CREATE2, so that a rerun finds the VM that an earlier run registered. AlphabetVM is a test
        // mock without an interface, so abi.encodeCall cannot encode its constructor arguments.
        output_.alphabetVM = AlphabetVM(
            // nosemgrep: sol-safety-deployutils-args
            DeployUtils.createDeterministic({
                _name: "AlphabetVM",
                _args: abi.encode(absolutePrestate, preimageOracle),
                _salt: DeployUtils.DEFAULT_SALT
            })
        );
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }

    function assertValidInput(Input memory _input) private pure {
        require(_input.absolutePrestate != bytes32(0), "DeployAlphabetVM: absolutePrestate not set");
        require(address(_input.preimageOracle) != address(0), "DeployAlphabetVM: preimageOracle not set");
    }

    function assertValidOutput(Output memory _output) private pure {
        require(address(_output.alphabetVM) != address(0), "DeployAlphabetVM: alphabetVM not set");
    }
}
