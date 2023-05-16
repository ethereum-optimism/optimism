// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";
import { ResourceMetering } from "../L1/ResourceMetering.sol";
import { Constants } from "../libraries/Constants.sol";

contract SystemConfig_Init is CommonTest {
    SystemConfig sysConf;

    function setUp() public virtual override {
        super.setUp();

        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            minimumBaseFee: 1 gwei,
            systemTxMaxGas: 1_000_000,
            maximumBaseFee: type(uint128).max
        });

        sysConf = new SystemConfig({
            _owner: alice,
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: 30_000_000,
            _unsafeBlockSigner: address(1),
            _config: config
        });
    }
}

contract SystemConfig_Initialize_TestFail is SystemConfig_Init {
    function test_initialize_lowGasLimit_reverts() external {
        uint64 minimumGasLimit = sysConf.minimumGasLimit();

        ResourceMetering.ResourceConfig memory cfg = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            minimumBaseFee: 1 gwei,
            systemTxMaxGas: 1_000_000,
            maximumBaseFee: type(uint128).max
        });

        vm.expectRevert("SystemConfig: gas limit too low");
        new SystemConfig({
            _owner: alice,
            _overhead: 0,
            _scalar: 0,
            _batcherHash: bytes32(hex""),
            _gasLimit: minimumGasLimit - 1,
            _unsafeBlockSigner: address(1),
            _config: cfg
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

    function test_setResourceConfig_notOwner_reverts() external {
        ResourceMetering.ResourceConfig memory config = Constants.DEFAULT_RESOURCE_CONFIG();
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setResourceConfig(config);
    }

    function test_setResourceConfig_badMinMax_reverts() external {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 2 gwei,
            maximumBaseFee: 1 gwei
        });
        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: min base fee must be less than max base");
        sysConf.setResourceConfig(config);
    }

    function test_setResourceConfig_zeroDenominator_reverts() external {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 0,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 1 gwei,
            maximumBaseFee: 2 gwei
        });
        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: denominator must be larger than 1");
        sysConf.setResourceConfig(config);
    }

    function test_setResourceConfig_lowGasLimit_reverts() external {
        uint64 gasLimit = sysConf.gasLimit();

        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: uint32(gasLimit),
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: uint32(gasLimit),
            minimumBaseFee: 1 gwei,
            maximumBaseFee: 2 gwei
        });
        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: gas limit too low");
        sysConf.setResourceConfig(config);
    }

    function test_setResourceConfig_badPrecision_reverts() external {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 11,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 1 gwei,
            maximumBaseFee: 2 gwei
        });
        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: precision loss with target resource limit");
        sysConf.setResourceConfig(config);
    }

    function testFuzz_setAttestor_nonOwner_reverts(address _attestor, bool _authenticated)
        external
    {
        vm.assume(_attestor != sysConf.owner());
        vm.prank(_attestor);
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setAttestor(_attestor, _authenticated);
    }

    function test_setAttestor_alreadyAuthenticated_reverts() external {
        vm.prank(sysConf.owner());
        sysConf.setAttestor(address(0x20), true);

        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: attestor already authenticated");
        sysConf.setAttestor(address(0x20), true);
    }

    function testFuzz_setAttestationThreshold_notOwner_reverts(address _caller) external {
        vm.assume(_caller != sysConf.owner());
        vm.prank(_caller);
        vm.expectRevert("Ownable: caller is not the owner");
        sysConf.setAttestationThreshold(0);
    }

    function test_setAttestationThreshold_zero_reverts() external {
        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: attestation threshold must be greater than 0");
        sysConf.setAttestationThreshold(0);
    }

    function testFuzz_setAttestationThreshold_exceedMax_reverts(uint256 _attestationThreshold)
        external
    {
        vm.assume(_attestationThreshold > 10_000);
        vm.prank(sysConf.owner());
        vm.expectRevert("SystemConfig: attestation threshold must not exceed 10,000");
        sysConf.setAttestationThreshold(_attestationThreshold);
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
        uint64 minimumGasLimit = sysConf.minimumGasLimit();
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

    function test_setAttestor_popSwap_succeeds() external {
        // Create a set of attestors
        vm.startPrank(sysConf.owner());
        sysConf.setAttestor(address(0x20), true);
        sysConf.setAttestor(address(0x21), true);
        sysConf.setAttestor(address(0x22), true);
        vm.stopPrank();

        // Remove the first attestor
        vm.prank(sysConf.owner());
        sysConf.setAttestor(address(0x20), false);

        // Validate the pop swap
        address[] memory attestorSet = sysConf.attestorSet();
        assertEq(attestorSet[0], address(0x22));
        assertEq(attestorSet[1], address(0x21));
    }

    function testFuzz_setAttestor_succeeds(address _attestor, bool _authenticated) external {
        address[] memory cachedAttestors = sysConf.attestorSet();
        uint256 len = cachedAttestors.length;
        for (uint256 i = 0; i < len; i++) {
            if (cachedAttestors[i] == _attestor) {
                if (_authenticated) {
                    vm.expectRevert("SystemConfig: attestor already authenticated");
                }
            }
        }
        vm.prank(sysConf.owner());
        sysConf.setAttestor(_attestor, _authenticated);
    }

    function testFuzz_setAttestationThreshold_succeeds(uint256 _attestationThreshold) external {
        vm.assume(_attestationThreshold > 0);
        vm.assume(_attestationThreshold <= 10_000);
        vm.prank(sysConf.owner());
        sysConf.setAttestationThreshold(_attestationThreshold);
    }
}
