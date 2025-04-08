// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { Claim } from "src/dispute/lib/Types.sol";

contract DeployAlphabetVM2 is Script {
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

        vm.broadcast(msg.sender);
        AlphabetVM alphabetVM = new AlphabetVM(absolutePrestate, preimageOracle);

        output_.alphabetVM = alphabetVM;
    }

    function assertValidInput(Input memory _input) private pure {
        require(_input.absolutePrestate != bytes32(0), "DeployAlphabetVM: absolutePrestate not set");
        require(address(_input.preimageOracle) != address(0), "DeployAlphabetVM: preimageOracle not set");
    }

    function assertValidOutput(Output memory _output) private pure {
        require(address(_output.alphabetVM) != address(0), "DeployAlphabetVM: alphabetVM not set");
    }
}
