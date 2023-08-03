// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { console } from "forge-std/console.sol";
import { console2 } from "forge-std/console2.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";

contract MIPS_Test is Test {
    MIPS internal mips;
    PreimageOracle internal oracle;

    function setUp() public {
        oracle = new PreimageOracle();
        mips = new MIPS();
        vm.store(address(mips), 0x0, bytes32(abi.encode(address(oracle))));
        vm.label(address(oracle), "PreimageOracle");
        vm.label(address(mips), "MIPS");
    }

    struct State {
        bytes32 one;
        bytes32 two;
    }

    function foo() public {
        unchecked {
            State memory h;
            uint256 val1;
            bool val = false;
            uint256 offset;
            assembly {
                let ptr := mload(0x40)
                val1 := ptr
                val := iszero(eq(h, 0x80))
                offset := h
            }
            console2.log("val is %s", val);
            console2.log("val1 is %s", val1);
            console2.log("offset is %s", offset);
            assertFalse(val, "mem offset check fail");
        }
    }

    function test_step_basics() external {
        foo();
        return;

        uint32[32] memory registers;
        MIPS.State memory state = MIPS.State({
            memRoot: bytes32(0),
            preimageKey: bytes32(0),
            preimageOffset: 0,
            pc: 100,
            nextPC: 104,
            lo: 0,
            hi: 0,
            heap: 0,
            exitCode: 0,
            exited: false,
            step: 0,
            registers: registers
        });
        bytes memory proof;

        bytes32 postState = mips.step(abi.encode(state), proof);
        assertTrue(postState != bytes32(0));
    }
}
