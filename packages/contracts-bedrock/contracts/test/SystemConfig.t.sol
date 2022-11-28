// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";

contract SystemConfig_Init is CommonTest {
    SystemConfig sysConf;

    function setUp() external {
        sysConf = new SystemConfig({
            _owner: alice,
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: 9_000_000
        });
    }
}

contract SystemConfig_Initialize_TestFail is CommonTest {
    function test_initialize_lowGasLimit_reverts() external {
        vm.expectRevert("SystemConfig: gas limit too low");
        new SystemConfig({
            _owner: alice,
            _overhead: 0,
            _scalar: 0,
            _batcherHash: bytes32(hex""),
            _gasLimit: 7_999_999
        });
    }
}

contract SystemConfig_Setters_TestFail is SystemConfig_Init {
    function test_setBatcherHash_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setBatcherHash(bytes32(hex""));
    }

    function test_setGasConfig_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setGasConfig(0, 0);
    }

    function test_setGasLimit_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setGasLimit(0);
    }
}

contract SystemConfig_Setters_Test is SystemConfig_Init {
    event ConfigUpdate(
        uint256 indexed version,
        SystemConfig.UpdateType indexed updateType,
        bytes data
    );

    function test_setBatcherHash_succeeds() external {
        bytes32 newBatcherHash = bytes32(hex"1234");

        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.BATCHER, abi.encode(newBatcherHash));

        vm.prank(alice);
        sysConf.setBatcherHash(newBatcherHash);
        assertEq(sysConf.batcherHash(), newBatcherHash);
    }

    function testFuzz_setBatcherHash_succeeds(bytes32 newBatcherHash) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.BATCHER, abi.encode(newBatcherHash));

        vm.prank(alice);
        sysConf.setBatcherHash(newBatcherHash);
        assertEq(sysConf.batcherHash(), newBatcherHash);
    }

    function test_setGasConfig_succeeds() external {
        uint256 newOverhead = 1234;
        uint256 newScalar = 5678;

        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(
            0,
            SystemConfig.UpdateType.GAS_CONFIG,
            abi.encode(newOverhead, newScalar)
        );

        vm.prank(alice);
        sysConf.setGasConfig(newOverhead, newScalar);
        assertEq(sysConf.overhead(), newOverhead);
        assertEq(sysConf.scalar(), newScalar);
    }

    function testFuzz_setGasConfig_succeeds(uint256 newOverhead, uint256 newScalar) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(
            0,
            SystemConfig.UpdateType.GAS_CONFIG,
            abi.encode(newOverhead, newScalar)
        );

        vm.prank(alice);
        sysConf.setGasConfig(newOverhead, newScalar);
        assertEq(sysConf.overhead(), newOverhead);
        assertEq(sysConf.scalar(), newScalar);
    }

    function test_setGasLimit_succeeds() external {
        uint64 newGasLimit = 9_876_543;

        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.GAS_LIMIT, abi.encode(newGasLimit));

        vm.prank(alice);
        sysConf.setGasLimit(newGasLimit);
        assertEq(sysConf.gasLimit(), newGasLimit);
    }

    function testFuzz_setGasLimit_succeeds(uint64 newGasLimit) external {
        uint64 minimumGasLimit = sysConf.MINIMUM_GAS_LIMIT();
        newGasLimit = uint64(
            bound(uint256(newGasLimit), uint256(minimumGasLimit), uint256(type(uint64).max))
        );

        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.GAS_LIMIT, abi.encode(newGasLimit));

        vm.prank(alice);
        sysConf.setGasLimit(newGasLimit);
        assertEq(sysConf.gasLimit(), newGasLimit);
    }
}
