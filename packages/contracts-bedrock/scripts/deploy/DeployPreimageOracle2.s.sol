// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

contract DeployPreimageOracle2 is Script {
    struct Input {
        uint256 minProposalSize;
        uint256 challengePeriod;
    }

    struct Output {
        IPreimageOracle preimageOracle;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployPreimageOracle(_input, output_);

        assertValidOutput(_input, output_);
    }

    function deployPreimageOracle(Input memory _input, Output memory _output) internal {
        vm.broadcast(msg.sender);
        IPreimageOracle preimageOracle = IPreimageOracle(
            DeployUtils.create1({
                _name: "PreimageOracle",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IPreimageOracle.__constructor__, (_input.minProposalSize, _input.challengePeriod))
                )
            })
        );

        vm.label(address(preimageOracle), "PreimageOracle");
        _output.preimageOracle = preimageOracle;
    }

    function assertValidInput(Input memory _input) internal pure {
        require(_input.minProposalSize != 0, "DeployPreimageOracle: minProposalSize not set");
        require(_input.challengePeriod != 0, "DeployPreimageOracle: challengePeriod not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal view {
        DeployUtils.assertValidContractAddress(address(_output.preimageOracle));
        require(_output.preimageOracle.minProposalSize() == _input.minProposalSize, "DPO-10");
        require(_output.preimageOracle.challengePeriod() == _input.challengePeriod, "DPO-20");
    }
}
