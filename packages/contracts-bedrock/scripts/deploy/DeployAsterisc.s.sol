// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IRISCV } from "interfaces/vendor/asterisc/IRISCV.sol";

/// @title DeployAsterisc
contract DeployAsterisc is Script {
    struct Input {
        IPreimageOracle preimageOracle;
    }

    struct Output {
        IRISCV asteriscSingleton;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployAsteriscSingleton(_input, output_);

        assertValidOutput(_input, output_);
    }

    function deployAsteriscSingleton(Input memory _input, Output memory _output) internal {
        vm.broadcast(msg.sender);
        IRISCV singleton = IRISCV(
            DeployUtils.create1({
                _name: "RISCV",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IRISCV.__constructor__, (_input.preimageOracle)))
            })
        );

        vm.label(address(singleton), "AsteriscSingleton");
        _output.asteriscSingleton = singleton;
    }

    function assertValidInput(Input memory _input) internal pure {
        require(address(_input.preimageOracle) != address(0), "DeployAsterisc: preimageOracle not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal view {
        DeployUtils.assertValidContractAddress(address(_output.asteriscSingleton));

        require(
            _output.asteriscSingleton.oracle() == _input.preimageOracle,
            "DeployAsterisc: preimageOracle does not match input"
        );
    }
}
