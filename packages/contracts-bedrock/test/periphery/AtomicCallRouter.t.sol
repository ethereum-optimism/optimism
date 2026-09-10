// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { AtomicCallRouter } from "src/periphery/AtomicCallRouter.sol";

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
