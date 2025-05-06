// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DeployAltDA } from "scripts/deploy/DeployAltDA.s.sol";
import { IDataAvailabilityChallenge } from "interfaces/L1/IDataAvailabilityChallenge.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

contract DeployAltDA_Test is Test {
    DeployAltDA deployAltDA;

    // Define defaults
    bytes32 salt = bytes32(uint256(1));
    IProxyAdmin proxyAdmin;
    address challengeContractOwner = makeAddr("challengeContractOwner");
    uint256 challengeWindow = 100;
    uint256 resolveWindow = 200;
    uint256 bondSize = 1 ether;
    uint256 resolverRefundPercentage = 10;

    function setUp() public {
        deployAltDA = new DeployAltDA();

        // Setup proxyAdmin
        proxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (msg.sender)))
            })
        );
    }

    function test_run_succeeds(
        DeployAltDA.Input memory _input,
        uint8 _resolverRefundPercentage // we use uint8 for a percentage value so that we don't need to reject almost
            // every uint256
    )
        public
    {
        vm.assume(_input.resolverRefundPercentage != 0);
        vm.assume(_resolverRefundPercentage <= 100);
        _input.resolverRefundPercentage = resolverRefundPercentage;

        vm.assume(_input.salt != bytes32(0));
        vm.assume(address(_input.proxyAdmin) != address(0));
        vm.assume(_input.challengeContractOwner != address(0));
        vm.assume(_input.challengeWindow != 0);
        vm.assume(_input.resolveWindow != 0);
        vm.assume(_input.bondSize != 0);

        // Run deployment
        DeployAltDA.Output memory output = deployAltDA.run(_input);

        // Verify everything is set up correctly
        assertTrue(address(output.dataAvailabilityChallengeImpl).code.length > 0, "200");

        IDataAvailabilityChallenge dac = output.dataAvailabilityChallengeProxy;
        assertTrue(address(dac).code.length > 0, "100");
        assertEq(dac.owner(), _input.challengeContractOwner, "300");
        assertEq(dac.challengeWindow(), _input.challengeWindow, "400");
        assertEq(dac.resolveWindow(), _input.resolveWindow, "500");
        assertEq(dac.bondSize(), _input.bondSize, "600");
        assertEq(dac.resolverRefundPercentage(), _input.resolverRefundPercentage, "700");

        // Make sure the proxy admin is set correctly.
        vm.prank(address(0));
        assertEq(IProxy(payable(address(dac))).admin(), address(_input.proxyAdmin), "800");
    }

    function test_run_resolverRefundPercentageTooLarge_reverts(uint256 _resolverRefundPercentage) public {
        vm.assume(_resolverRefundPercentage > 100);

        DeployAltDA.Input memory input = defaultInput();
        input.resolverRefundPercentage = _resolverRefundPercentage;

        vm.expectRevert("DeployAltDA: resolverRefundPercentage too large");
        deployAltDA.run(input);
    }

    function test_run_nullInputs_reverts() public {
        DeployAltDA.Input memory input;

        input = defaultInput();
        input.salt = bytes32(0);
        vm.expectRevert("DeployAltDA: salt not set");
        deployAltDA.run(input);

        input = defaultInput();
        input.proxyAdmin = IProxyAdmin(address(0));
        vm.expectRevert("DeployAltDA: proxyAdmin not set");
        deployAltDA.run(input);

        input = defaultInput();
        input.challengeContractOwner = address(0);
        vm.expectRevert("DeployAltDA: challengeContractOwner not set");
        deployAltDA.run(input);

        input = defaultInput();
        input.challengeWindow = 0;
        vm.expectRevert("DeployAltDA: challengeWindow not set");
        deployAltDA.run(input);

        input = defaultInput();
        input.resolveWindow = 0;
        vm.expectRevert("DeployAltDA: resolveWindow not set");
        deployAltDA.run(input);

        input = defaultInput();
        input.bondSize = 0;
        vm.expectRevert("DeployAltDA: bondSize not set");
        deployAltDA.run(input);
    }

    function defaultInput() private view returns (DeployAltDA.Input memory input_) {
        input_ = DeployAltDA.Input(
            salt, proxyAdmin, challengeContractOwner, challengeWindow, resolveWindow, bondSize, resolverRefundPercentage
        );
    }
}
