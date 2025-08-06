// SPDX-License-Identifier: MIT.
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

import { IProxy } from "interfaces/universal/IProxy.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { Timelock } from "src/L1/Timelock.sol";

// Sample invocation:
//   forge script scripts/deploy/DeployTimelock.s.sol \
//     --sig "run((address[],uint64,uint64))" \
//     "([0x0000000000000000000000000000000000000001,0x0000000000000000000000000000000000000002], 20, 10)"
contract DeployTimelock is Script {
    struct Input {
        address[] controllers;
        uint64 longDelay;
        uint64 shortDelay;
    }

    struct Output {
        Timelock timelockImpl;
        Timelock timelockProxy;
    }

    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployTimelock(_input, output_);

        assertValidOutput(_input, output_);
    }

    function assertValidInput(Input memory _input) internal pure {
        // All required checks are in the timelock, so this is a no-op.
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal view {
        DeployUtils.assertValidContractAddress(address(_output.timelockImpl));
        DeployUtils.assertValidContractAddress(address(_output.timelockProxy));
    }

    function deployTimelock(Input memory _input, Output memory _output) public {
        bytes32 salt = getSalt(_input);
        address deployer = msg.sender;

        vm.startBroadcast(deployer);

        // Deploy the timelock implementation.
        address timelockImpl = DeployUtils.create2({ _name: "Timelock", _args: hex"", _salt: salt });

        // Deploy the timelock proxy, with deployer as the admin.
        bytes memory args = DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (deployer)));
        address timelockProxy = DeployUtils.create2({ _name: "Proxy", _args: args, _salt: salt });

        // Set and initialize the timelock implementation.
        IProxy(payable(timelockProxy)).upgradeToAndCall(
            timelockImpl, abi.encodeCall(Timelock.initialize, (_input.controllers, _input.longDelay, _input.shortDelay))
        );

        // Transfer ownership of the timelock proxy to itself.
        IProxy(payable(timelockProxy)).changeAdmin(timelockProxy);

        vm.stopBroadcast();

        _output.timelockImpl = Timelock(timelockImpl);
        _output.timelockProxy = Timelock(timelockProxy);
    }

    function getSalt(Input memory _input) internal pure returns (bytes32) {
        return keccak256(abi.encode(_input));
    }
}
