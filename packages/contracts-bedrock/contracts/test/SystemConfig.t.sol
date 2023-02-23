// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";

contract SystemConfig_Init is CommonTest {
    SystemConfig sysConf;

    function setUp() public virtual override {
        super.setUp();
        sysConf = new SystemConfig({
            _owner: alice,
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: 9_000_000,
            _unsafeBlockSigner: address(1)
        });
    }
}

contract SystemConfig_Initialize_TestFail is CommonTest {
    function test_initialize_lowGasLimit_reverts() external {
        vm.expectRevert("SystemConfig: gas limit too low");

        // The minimum gas limit defined in SystemConfig:
        uint64 MINIMUM_GAS_LIMIT = 8_000_000;
        new SystemConfig({
            _owner: alice,
            _overhead: 0,
            _scalar: 0,
            _batcherHash: bytes32(hex""),
            _gasLimit: MINIMUM_GAS_LIMIT - 1,
            _unsafeBlockSigner: address(1)
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

    function test_setUnsafeBlockSigner_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setUnsafeBlockSigner(address(0x20));
    }
}

contract SystemConfig_Setters_Test is SystemConfig_Init {
    event ConfigUpdate(
        uint256 indexed version,
        SystemConfig.UpdateType indexed updateType,
        bytes data
    );

    function testFuzz_setBatcherHash_succeeds(bytes32 newBatcherHash) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.BATCHER, abi.encode(newBatcherHash));

        vm.prank(sysConf.owner());
        sysConf.setBatcherHash(newBatcherHash);
        assertEq(sysConf.batcherHash(), newBatcherHash);
    }

    function testFuzz_setGasConfig_succeeds(uint256 newOverhead, uint256 newScalar) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(
            0,
            SystemConfig.UpdateType.GAS_CONFIG,
            abi.encode(newOverhead, newScalar)
        );

        vm.prank(sysConf.owner());
        sysConf.setGasConfig(newOverhead, newScalar);
        assertEq(sysConf.overhead(), newOverhead);
        assertEq(sysConf.scalar(), newScalar);
    }

    function testFuzz_setGasLimit_succeeds(uint64 newGasLimit) external {
        uint64 minimumGasLimit = sysConf.MINIMUM_GAS_LIMIT();
        newGasLimit = uint64(
            bound(uint256(newGasLimit), uint256(minimumGasLimit), uint256(type(uint64).max))
        );

        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.GAS_LIMIT, abi.encode(newGasLimit));

        vm.prank(sysConf.owner());
        sysConf.setGasLimit(newGasLimit);
        assertEq(sysConf.gasLimit(), newGasLimit);
    }

    function testFuzz_setUnsafeBlockSigner_succeeds(address newUnsafeSigner) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(
            0,
            SystemConfig.UpdateType.UNSAFE_BLOCK_SIGNER,
            abi.encode(newUnsafeSigner)
        );

        vm.prank(sysConf.owner());
        sysConf.setUnsafeBlockSigner(newUnsafeSigner);
        assertEq(sysConf.unsafeBlockSigner(), newUnsafeSigner);
    }
}
