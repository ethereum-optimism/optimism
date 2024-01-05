// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { SafeCall } from "src/libraries/SafeCall.sol";

contract SafeCall_Test is Test {
    /// @dev Tests that the `send` function succeeds.
    function testFuzz_send_succeeds(address from, address to, uint256 gas, uint64 value) external {
        vm.assume(from.balance == 0);
        vm.assume(to.balance == 0);
        // no precompiles (mainnet)
        assumeNotPrecompile(to);
        // don't call the vm
        vm.assume(to != address(vm));
        vm.assume(from != address(vm));
        // don't call the console
        vm.assume(to != address(0x000000000000000000636F6e736F6c652e6c6f67));
        // don't call the create2 deployer
        vm.assume(to != address(0x4e59b44847b379578588920cA78FbF26c0B4956C));
        vm.assume(to != address(this));

        assertEq(from.balance, 0, "from balance is 0");
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");

        uint256[2] memory balancesBefore = [from.balance, to.balance];

        vm.expectCall(to, value, bytes(""));
        vm.prank(from);
        bool success = SafeCall.send(to, gas, value);

        assertTrue(success, "send not successful");
        if (from == to) {
            assertEq(from.balance, balancesBefore[0], "Self-send did not change balance");
        } else {
            assertEq(from.balance, balancesBefore[0] - value, "from balance not drained");
            assertEq(to.balance, balancesBefore[1] + value, "to balance received");
        }
    }

    /// @dev Tests that `call` succeeds.
    function testFuzz_call_succeeds(address from, address to, uint256 gas, uint64 value, bytes memory data) external {
        vm.assume(from.balance == 0);
        vm.assume(to.balance == 0);
        // no precompiles (mainnet)
        assumeNotPrecompile(to);
        // don't call the vm
        vm.assume(to != address(vm));
        vm.assume(from != address(vm));
        // don't call the console
        vm.assume(to != address(0x000000000000000000636F6e736F6c652e6c6f67));
        // don't call the create2 deployer
        vm.assume(to != address(0x4e59b44847b379578588920cA78FbF26c0B4956C));
        vm.assume(to != address(this));

        assertEq(from.balance, 0, "from balance is 0");
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");

        uint256[2] memory balancesBefore = [from.balance, to.balance];

        vm.expectCall(to, value, data);
        vm.prank(from);
        bool success = SafeCall.call(to, gas, value, data);

        assertTrue(success, "call not successful");
        if (from == to) {
            assertEq(from.balance, balancesBefore[0], "Self-send did not change balance");
        } else {
            assertEq(from.balance, balancesBefore[0] - value, "from balance not drained");
            assertEq(to.balance, balancesBefore[1] + value, "to balance received");
        }
    }

    /// @dev Tests that `callWithMinGas` succeeds with enough gas.
    function testFuzz_callWithMinGas_hasEnough_succeeds(
        address from,
        address to,
        uint64 minGas,
        uint64 value,
        bytes memory data
    )
        external
    {
        vm.assume(from.balance == 0);
        vm.assume(to.balance == 0);
        // no precompiles (mainnet)
        assumeNotPrecompile(to);
        // don't call the vm
        vm.assume(to != address(vm));
        vm.assume(from != address(vm));
        // don't call the console
        vm.assume(to != address(0x000000000000000000636F6e736F6c652e6c6f67));
        // don't call the create2 deployer
        vm.assume(to != address(0x4e59b44847b379578588920cA78FbF26c0B4956C));
        vm.assume(to != address(this));

        assertEq(from.balance, 0, "from balance is 0");
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");

        // Bound minGas to [0, l1_block_gas_limit]
        minGas = uint64(bound(minGas, 0, 30_000_000));

        uint256[2] memory balancesBefore = [from.balance, to.balance];

        vm.expectCallMinGas(to, value, minGas, data);
        vm.prank(from);
        bool success = SafeCall.callWithMinGas(to, minGas, value, data);

        assertTrue(success, "call not successful");
        if (from == to) {
            assertEq(from.balance, balancesBefore[0], "Self-send did not change balance");
        } else {
            assertEq(from.balance, balancesBefore[0] - value, "from balance not drained");
            assertEq(to.balance, balancesBefore[1] + value, "to balance received");
        }
    }

    /// @dev Tests that `callWithMinGas` succeeds for the lower gas bounds.
    function test_callWithMinGas_noLeakageLow_succeeds() external {
        SimpleSafeCaller caller = new SimpleSafeCaller();

        for (uint64 i = 40_000; i < 100_000; i++) {
            uint256 snapshot = vm.snapshot();

            // 65_907 is the exact amount of gas required to make the safe call
            // successfully.
            if (i < 65_907) {
                assertFalse(caller.makeSafeCall(i, 25_000));
            } else {
                vm.expectCallMinGas(address(caller), 0, 25_000, abi.encodeWithSelector(caller.setA.selector, 1));
                assertTrue(caller.makeSafeCall(i, 25_000));
            }

            assertTrue(vm.revertTo(snapshot));
        }
    }

    /// @dev Tests that `callWithMinGas` succeeds on the upper gas bounds.
    function test_callWithMinGas_noLeakageHigh_succeeds() external {
        SimpleSafeCaller caller = new SimpleSafeCaller();

        for (uint64 i = 15_200_000; i < 15_300_000; i++) {
            uint256 snapshot = vm.snapshot();

            // 15_278_606 is the exact amount of gas required to make the safe call
            // successfully.
            if (i < 15_278_606) {
                assertFalse(caller.makeSafeCall(i, 15_000_000));
            } else {
                vm.expectCallMinGas(address(caller), 0, 15_000_000, abi.encodeWithSelector(caller.setA.selector, 1));
                assertTrue(caller.makeSafeCall(i, 15_000_000));
            }

            assertTrue(vm.revertTo(snapshot));
        }
    }
}

contract SimpleSafeCaller {
    uint256 public a;

    function makeSafeCall(uint64 gas, uint64 minGas) external returns (bool) {
        return SafeCall.call(address(this), gas, 0, abi.encodeWithSelector(this.makeSafeCallMinGas.selector, minGas));
    }

    function makeSafeCallMinGas(uint64 minGas) external returns (bool) {
        return SafeCall.callWithMinGas(address(this), minGas, 0, abi.encodeWithSelector(this.setA.selector, 1));
    }

    function setA(uint256 _a) external {
        a = _a;
    }
}
