// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { LibString } from "@solady/utils/LibString.sol";

// Interfaces
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

import { DeployDelayedWETH2 } from "scripts/deploy/DeployDelayedWETH2.s.sol";

contract DeployDelayedWETH2_Test is Test {
    DeployDelayedWETH2 deployDelayedWETH;

    // Define default input variables for testing.
    string defaultRelease = "v1.0.0";
    address defaultProxyAdmin = makeAddr("defaultProxyAdmin");
    ISuperchainConfig defaultSuperchainConfigProxy = ISuperchainConfig(makeAddr("superchainConfigProxy"));
    address defaultDelayedWethOwner = makeAddr("delayedWethOwner");
    uint256 defaultDelayedWethDelay = 1 days;

    function setUp() public {
        deployDelayedWETH = new DeployDelayedWETH2();
    }

    function testFuzz_run_withoutDeployedImpl_succeeds(DeployDelayedWETH2.Input memory _input) public {
        vm.assume(_input.proxyAdmin != address(0));
        vm.assume(address(_input.superchainConfigProxy) != address(0));
        vm.assume(_input.delayedWethDelay != 0);
        vm.assume(_input.delayedWethOwner != address(0));
        vm.assume(!LibString.eq(_input.release, ""));

        // make sure we don't pass implementation in
        _input.delayedWethImpl == address(0);

        // Run the deployment script.
        deployDelayedWETH.run(_input);
    }

    function testFuzz_run_withDeployedImplAndDevelopRelease_succeeds(DeployDelayedWETH2.Input memory _input) public {
        vm.assume(_input.proxyAdmin != address(0));
        vm.assume(address(_input.superchainConfigProxy) != address(0));
        vm.assume(_input.delayedWethDelay != 0);
        vm.assume(_input.delayedWethOwner != address(0));
        vm.assume(!LibString.eq(_input.release, ""));
        vm.assume(!LibString.startsWith(_input.release, "op-contracts"));

        // predeploy the implementation
        _input.delayedWethImpl = DeployUtils.create1({
            _name: "DelayedWETH",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IDelayedWETH.__constructor__, (_input.delayedWethDelay)))
        });

        // Run the deployment script.
        deployDelayedWETH.run(_input);
    }

    function testFuzz_run_withoutDeployedImplAndNonDevelopRelease_reverts(
        DeployDelayedWETH2.Input memory _input,
        string memory _releaseSuffix
    )
        public
    {
        vm.assume(_input.proxyAdmin != address(0));
        vm.assume(address(_input.superchainConfigProxy) != address(0));
        vm.assume(_input.delayedWethDelay != 0);
        vm.assume(_input.delayedWethOwner != address(0));

        // make sure the release starts with op-contracts
        _input.release = string.concat("op-contracts", _releaseSuffix);

        // clear the implementation
        _input.delayedWethImpl = address(0);

        // Run the deployment script.
        DeployDelayedWETH2_Test(this).helper_run_withoutDeployedImplAndNonDevelopRelease_reverts(
            string.concat("DeployDelayedWETH: failed to deploy release ", _input.release), _input
        );
    }

    function helper_run_withoutDeployedImplAndNonDevelopRelease_reverts(
        string calldata _message,
        DeployDelayedWETH2.Input memory _input
    )
        external
    {
        vm.expectRevert(bytes(_message));
        deployDelayedWETH.run(_input);
    }

    function test_run_nullInput_reverts() public {
        DeployDelayedWETH2.Input memory input;

        input = defaultInput();
        input.release = "";
        vm.expectRevert("DeployDelayedWETH: release not set");
        deployDelayedWETH.run(input);

        input = defaultInput();
        input.proxyAdmin = address(0);
        vm.expectRevert("DeployDelayedWETH: proxyAdmin not set");
        deployDelayedWETH.run(input);

        input = defaultInput();
        input.superchainConfigProxy = ISuperchainConfig(address(0));
        vm.expectRevert("DeployDelayedWETH: superchainConfigProxy not set");
        deployDelayedWETH.run(input);

        input = defaultInput();
        input.delayedWethOwner = address(0);
        vm.expectRevert("DeployDelayedWETH: delayedWethOwner not set");
        deployDelayedWETH.run(input);

        input = defaultInput();
        input.delayedWethDelay = 0;
        vm.expectRevert("DeployDelayedWETH: delayedWethDelay not set");
        deployDelayedWETH.run(input);
    }

    function defaultInput() internal view returns (DeployDelayedWETH2.Input memory input_) {
        input_ = DeployDelayedWETH2.Input(
            defaultRelease,
            defaultProxyAdmin,
            defaultSuperchainConfigProxy,
            address(0), // delayedWethImpl
            defaultDelayedWethOwner,
            defaultDelayedWethDelay
        );
    }
}
