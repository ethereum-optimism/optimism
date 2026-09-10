// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { AtomicCallRouter } from "src/periphery/AtomicCallRouter.sol";

import {
    AtomicResultWitness,
    AtomicRemoteCall,
    AtomicWitnessRequest,
    AtomicStreamCursor
} from "src/libraries/AtomicCallTypes.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { IAtomicCounter } from "interfaces/integration/IAtomicCounter.sol";

abstract contract AtomicCallRouter_TestInit is Test {
    AtomicCallRouter internal router;
    uint256 internal constant REMOTE_CHAIN = 902;
    address internal constant TARGET = address(128);

    function setUp() public {
        vm.chainId(901);
        router = new AtomicCallRouter();
    }
}

contract AtomicCallRouter_PredictProxy_Test is AtomicCallRouter_TestInit {
    function test_predictProxy_matchesDeployment_succeeds() external {
        address predicted = router.predictProxy(REMOTE_CHAIN, TARGET);
        assertEq(predicted.code.length, 0, "prediction does not deploy the facade");
        assertEq(router.proxyFor(REMOTE_CHAIN, TARGET), predicted);
        assertGt(predicted.code.length, 0);
        assertTrue(router.proxies(predicted));
    }

    function test_predictProxy_distinctPeers_succeeds() external view {
        address predicted = router.predictProxy(REMOTE_CHAIN, TARGET);
        assertNotEq(predicted, router.predictProxy(REMOTE_CHAIN + 1, TARGET));
        assertNotEq(predicted, router.predictProxy(REMOTE_CHAIN, address(256)));
    }

    function test_predictProxy_localChain_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InvalidPeer.selector);
        router.predictProxy(block.chainid, TARGET);
    }

    function test_predictProxy_zeroTarget_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InvalidPeer.selector);
        router.predictProxy(REMOTE_CHAIN, address(0));
    }
}

contract AtomicCallRouter_ProxyFor_Test is AtomicCallRouter_TestInit {
    function test_proxyFor_repeatedDeployment_succeeds() external {
        address deployed = router.proxyFor(REMOTE_CHAIN, TARGET);
        bytes32 codeHash = deployed.codehash;
        vm.recordLogs();
        assertEq(router.proxyFor(REMOTE_CHAIN, TARGET), deployed);
        assertEq(deployed.codehash, codeHash);
        assertTrue(router.proxies(deployed));
        assertEq(vm.getRecordedLogs().length, 0, "existing facade is returned without redeployment");
    }
}

/// @notice Gas-sensitive application fixture for the suspended REVM integration test.
contract AtomicCallRouter_GasProbe_Harness {
    uint256 public beforeGas;
    uint256 public afterFirstGas;
    uint256 public afterSecondGas;
    uint256 public blockLimit;
    uint256 public result;

    function run(address _proxy, uint256 _amount) external returns (uint256 result_) {
        beforeGas = gasleft();
        blockLimit = block.gaslimit;
        assembly {
            tstore(0, 123)
        }
        uint256 first = IAtomicCounter(_proxy).add(_amount);
        afterFirstGas = gasleft();
        uint256 retained;
        assembly {
            retained := tload(0)
        }
        require(retained == 123 && beforeGas > gasleft(), "AtomicCallRouter: frame lost");
        result_ = IAtomicCounter(_proxy).add(first);
        afterSecondGas = gasleft();
        result = result_;
    }

    function gasAtEntry() external view returns (uint256 gas_) {
        return gasleft();
    }
}

contract AtomicCallRouter_Uncategorized_Test is Test {
    AtomicCallRouter internal router;
    AtomicCallRouter_GasProbe_Harness internal probe;

    function setUp() public {
        router = new AtomicCallRouter();
        probe = new AtomicCallRouter_GasProbe_Harness();
    }

    function test_executeRootWithGas_sameApplicationBudget_succeeds() external {
        AtomicResultWitness[] memory witnesses = new AtomicResultWitness[](0);
        bytes memory data = abi.encodeCall(probe.gasAtEntry, ());
        bytes memory first = router.executeRootWithGas{ gas: 1_000_000 }(0, address(probe), data, witnesses, 100_000);
        bytes memory second = router.executeRootWithGas{ gas: 2_000_000 }(1, address(probe), data, witnesses, 100_000);
        assertEq(abi.decode(first, (uint256)), abi.decode(second, (uint256)));
    }

    function test_executeRootWithGas_insufficientOuterGas_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InsufficientGas.selector);
        router.executeRootWithGas{ gas: 200_000 }(
            0, address(probe), abi.encodeCall(probe.gasAtEntry, ()), new AtomicResultWitness[](0), 200_000
        );
    }

    function test_executeRootWithGas_largeCalldataBudget_succeeds() external {
        bytes memory data = abi.encodePacked(probe.gasAtEntry.selector, new bytes(160_000));
        bytes memory baseline =
            router.executeRootWithGas(0, address(probe), data, new AtomicResultWitness[](0), 200_000);
        uint256 expected = abi.decode(baseline, (uint256));
        uint256 succeeded;
        uint256 rejected;
        for (uint256 outer = 250_000; outer < 1_500_000; outer += 50_000) {
            bytes memory callData = abi.encodeCall(
                router.executeRootWithGas,
                (router.nonces(address(this)), address(probe), data, new AtomicResultWitness[](0), 200_000)
            );
            // eip150-safe: every success must receive the full application gas budget;
            // deliberately underfunded calls are expected to fail at the router.
            (bool success, bytes memory output) = address(router).call{ gas: outer }(callData);
            if (success) {
                assertEq(abi.decode(abi.decode(output, (bytes)), (uint256)), expected);
                succeeded++;
            } else {
                rejected++;
            }
        }
        assertGt(succeeded, 0);
        assertGt(rejected, 0);
    }

    function test_executeRootWithGas_zeroBudget_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InsufficientGas.selector);
        router.executeRootWithGas(0, address(probe), "", new AtomicResultWitness[](0), 0);
    }

    function test_executeRemoteWithGas_overCapacity_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_CallLimit.selector);
        router.executeRemoteWithGas(
            bytes32(0),
            new AtomicRemoteCall[](2),
            new AtomicResultWitness[](0),
            Identifier(address(0), 0, 0, 0, 0),
            100_000,
            1
        );
    }

    function test_witnessAt_externalReader_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InvalidReader.selector);
        router.witnessAt(AtomicWitnessRequest(0, 0, address(0), address(0), ""));
    }

    function test_remoteCallAt_externalReader_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InvalidReader.selector);
        router.remoteCallAt(AtomicStreamCursor(0, ""));
    }

    function test_witnessCount_externalReader_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InvalidReader.selector);
        router.witnessCount();
    }

    function test_completionIdentifier_externalReader_reverts() external {
        vm.expectRevert(AtomicCallRouter.AtomicCallRouter_InvalidReader.selector);
        router.completionIdentifier();
    }
}
