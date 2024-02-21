// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Target contract dependencies
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Target contract
import { SystemConfig } from "src/L1/SystemConfig.sol";

contract SystemConfig_Init is CommonTest {
    event ConfigUpdate(uint256 indexed version, SystemConfig.UpdateType indexed updateType, bytes data);
}

contract SystemConfig_Initialize_Test is SystemConfig_Init {
    address batchInbox;
    address owner;
    uint256 overhead;
    uint256 scalar;
    bytes32 batcherHash;
    uint64 gasLimit;
    address unsafeBlockSigner;
    address systemConfigImpl;
    address optimismMintableERC20Factory;

    function setUp() public virtual override {
        super.setUp();
        batchInbox = deploy.cfg().batchInboxAddress();
        owner = deploy.cfg().finalSystemOwner();
        overhead = deploy.cfg().gasPriceOracleOverhead();
        scalar = deploy.cfg().gasPriceOracleScalar();
        batcherHash = bytes32(uint256(uint160(deploy.cfg().batchSenderAddress())));
        gasLimit = uint64(deploy.cfg().l2GenesisBlockGasLimit());
        unsafeBlockSigner = deploy.cfg().p2pSequencerAddress();
        systemConfigImpl = deploy.mustGetAddress("SystemConfig");
        optimismMintableERC20Factory = deploy.mustGetAddress("OptimismMintableERC20FactoryProxy");
    }

    /// @dev Tests that constructor sets the correct values.
    function test_constructor_succeeds() external {
        SystemConfig impl = SystemConfig(systemConfigImpl);
        assertEq(impl.owner(), address(0xdEaD));
        assertEq(impl.overhead(), 0);
        assertEq(impl.scalar(), 0);
        assertEq(impl.batcherHash(), bytes32(0));
        assertEq(impl.gasLimit(), 1);
        assertEq(impl.unsafeBlockSigner(), address(0));
        ResourceMetering.ResourceConfig memory actual = impl.resourceConfig();
        assertEq(actual.maxResourceLimit, 1);
        assertEq(actual.elasticityMultiplier, 1);
        assertEq(actual.baseFeeMaxChangeDenominator, 2);
        assertEq(actual.minimumBaseFee, 0);
        assertEq(actual.systemTxMaxGas, 0);
        assertEq(actual.maximumBaseFee, 0);
        assertEq(impl.startBlock(), type(uint256).max);
        assertEq(address(impl.batchInbox()), address(0));
        // Check addresses
        assertEq(address(impl.l1CrossDomainMessenger()), address(0));
        assertEq(address(impl.l1ERC721Bridge()), address(0));
        assertEq(address(impl.l1StandardBridge()), address(0));
        assertEq(address(impl.l2OutputOracle()), address(0));
        assertEq(address(impl.optimismPortal()), address(0));
        assertEq(address(impl.optimismMintableERC20Factory()), address(0));
    }

    /// @dev Tests that initailization sets the correct values.
    function test_initialize_succeeds() external {
        assertEq(systemConfig.owner(), owner);
        assertEq(systemConfig.overhead(), overhead);
        assertEq(systemConfig.scalar(), scalar);
        assertEq(systemConfig.batcherHash(), batcherHash);
        assertEq(systemConfig.gasLimit(), gasLimit);
        assertEq(systemConfig.unsafeBlockSigner(), unsafeBlockSigner);
        // Depends on `initialize` being called with defaults
        ResourceMetering.ResourceConfig memory rcfg = Constants.DEFAULT_RESOURCE_CONFIG();
        ResourceMetering.ResourceConfig memory actual = systemConfig.resourceConfig();
        assertEq(actual.maxResourceLimit, rcfg.maxResourceLimit);
        assertEq(actual.elasticityMultiplier, rcfg.elasticityMultiplier);
        assertEq(actual.baseFeeMaxChangeDenominator, rcfg.baseFeeMaxChangeDenominator);
        assertEq(actual.minimumBaseFee, rcfg.minimumBaseFee);
        assertEq(actual.systemTxMaxGas, rcfg.systemTxMaxGas);
        assertEq(actual.maximumBaseFee, rcfg.maximumBaseFee);
        // Depends on start block being set to 0 in `initialize`
        uint256 cfgStartBlock = deploy.cfg().systemConfigStartBlock();
        assertEq(systemConfig.startBlock(), (cfgStartBlock == 0 ? block.number : cfgStartBlock));
        assertEq(address(systemConfig.batchInbox()), address(batchInbox));
        // Check addresses
        assertEq(address(systemConfig.l1CrossDomainMessenger()), address(l1CrossDomainMessenger));
        assertEq(address(systemConfig.l1ERC721Bridge()), address(l1ERC721Bridge));
        assertEq(address(systemConfig.l1StandardBridge()), address(l1StandardBridge));
        assertEq(address(systemConfig.l2OutputOracle()), address(l2OutputOracle));
        assertEq(address(systemConfig.optimismPortal()), address(optimismPortal));
        assertEq(address(systemConfig.optimismMintableERC20Factory()), address(optimismMintableERC20Factory));
    }
}

contract SystemConfig_Initialize_TestFail is SystemConfig_Initialize_Test {
    /// @dev Tests that initialization reverts if the gas limit is too low.
    function test_initialize_lowGasLimit_reverts() external {
        uint64 minimumGasLimit = systemConfig.minimumGasLimit();

        // Wipe out the initialized slot so the proxy can be initialized again
        vm.store(address(systemConfig), bytes32(0), bytes32(0));

        address admin = address(uint160(uint256(vm.load(address(systemConfig), Constants.PROXY_OWNER_ADDRESS))));
        vm.prank(admin);

        vm.expectRevert("SystemConfig: gas limit too low");
        systemConfig.initialize({
            _owner: alice,
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: minimumGasLimit - 1,
            _unsafeBlockSigner: address(1),
            _config: Constants.DEFAULT_RESOURCE_CONFIG(),
            _batchInbox: address(0),
            _addresses: SystemConfig.Addresses({
                l1CrossDomainMessenger: address(0),
                l1ERC721Bridge: address(0),
                l1StandardBridge: address(0),
                l2OutputOracle: address(0),
                optimismPortal: address(0),
                optimismMintableERC20Factory: address(0)
            })
        });
    }

    /// @dev Tests that startBlock is updated correctly when it's zero.
    function test_startBlock_update_succeeds() external {
        // Wipe out the initialized slot so the proxy can be initialized again
        vm.store(address(systemConfig), bytes32(0), bytes32(0));
        // Set slot startBlock to zero
        vm.store(address(systemConfig), systemConfig.START_BLOCK_SLOT(), bytes32(uint256(0)));

        // Initialize and check that StartBlock updates to current block number
        vm.prank(systemConfig.owner());
        systemConfig.initialize({
            _owner: alice,
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: gasLimit,
            _unsafeBlockSigner: address(1),
            _config: Constants.DEFAULT_RESOURCE_CONFIG(),
            _batchInbox: address(0),
            _addresses: SystemConfig.Addresses({
                l1CrossDomainMessenger: address(0),
                l1ERC721Bridge: address(0),
                l1StandardBridge: address(0),
                l2OutputOracle: address(0),
                optimismPortal: address(0),
                optimismMintableERC20Factory: address(0)
            })
        });
        assertEq(systemConfig.startBlock(), block.number);
    }

    /// @dev Tests that startBlock is not updated when it's not zero.
    function test_startBlock_update_fails() external {
        // Wipe out the initialized slot so the proxy can be initialized again
        vm.store(address(systemConfig), bytes32(0), bytes32(0));
        // Set slot startBlock to non-zero value 1
        vm.store(address(systemConfig), systemConfig.START_BLOCK_SLOT(), bytes32(uint256(1)));

        // Initialize and check that StartBlock doesn't update
        vm.prank(systemConfig.owner());
        systemConfig.initialize({
            _owner: alice,
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: gasLimit,
            _unsafeBlockSigner: address(1),
            _config: Constants.DEFAULT_RESOURCE_CONFIG(),
            _batchInbox: address(0),
            _addresses: SystemConfig.Addresses({
                l1CrossDomainMessenger: address(0),
                l1ERC721Bridge: address(0),
                l1StandardBridge: address(0),
                l2OutputOracle: address(0),
                optimismPortal: address(0),
                optimismMintableERC20Factory: address(0)
            })
        });
        assertEq(systemConfig.startBlock(), 1);
    }
}

contract SystemConfig_Setters_TestFail is SystemConfig_Init {
    /// @dev Tests that `setBatcherHash` reverts if the caller is not the owner.
    function test_setBatcherHash_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        systemConfig.setBatcherHash(bytes32(hex""));
    }

    /// @dev Tests that `setGasConfig` reverts if the caller is not the owner.
    function test_setGasConfig_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        systemConfig.setGasConfig(0, 0);
    }

    /// @dev Tests that `setGasLimit` reverts if the caller is not the owner.
    function test_setGasLimit_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        systemConfig.setGasLimit(0);
    }

    /// @dev Tests that `setUnsafeBlockSigner` reverts if the caller is not the owner.
    function test_setUnsafeBlockSigner_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        systemConfig.setUnsafeBlockSigner(address(0x20));
    }

    /// @dev Tests that `setResourceConfig` reverts if the caller is not the owner.
    function test_setResourceConfig_notOwner_reverts() external {
        ResourceMetering.ResourceConfig memory config = Constants.DEFAULT_RESOURCE_CONFIG();
        vm.expectRevert("Ownable: caller is not the owner");
        systemConfig.setResourceConfig(config);
    }

    /// @dev Tests that `setResourceConfig` reverts if the min base fee
    ///      is greater than the maximum allowed base fee.
    function test_setResourceConfig_badMinMax_reverts() external {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 2 gwei,
            maximumBaseFee: 1 gwei
        });
        vm.prank(systemConfig.owner());
        vm.expectRevert("SystemConfig: min base fee must be less than max base");
        systemConfig.setResourceConfig(config);
    }

    /// @dev Tests that `setResourceConfig` reverts if the baseFeeMaxChangeDenominator
    ///      is zero.
    function test_setResourceConfig_zeroDenominator_reverts() external {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 0,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 1 gwei,
            maximumBaseFee: 2 gwei
        });
        vm.prank(systemConfig.owner());
        vm.expectRevert("SystemConfig: denominator must be larger than 1");
        systemConfig.setResourceConfig(config);
    }

    /// @dev Tests that `setResourceConfig` reverts if the gas limit is too low.
    function test_setResourceConfig_lowGasLimit_reverts() external {
        uint64 gasLimit = systemConfig.gasLimit();

        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: uint32(gasLimit),
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: uint32(gasLimit),
            minimumBaseFee: 1 gwei,
            maximumBaseFee: 2 gwei
        });
        vm.prank(systemConfig.owner());
        vm.expectRevert("SystemConfig: gas limit too low");
        systemConfig.setResourceConfig(config);
    }

    /// @dev Tests that `setResourceConfig` reverts if the elasticity multiplier
    ///      and max resource limit are configured such that there is a loss of precision.
    function test_setResourceConfig_badPrecision_reverts() external {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 11,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 1 gwei,
            maximumBaseFee: 2 gwei
        });
        vm.prank(systemConfig.owner());
        vm.expectRevert("SystemConfig: precision loss with target resource limit");
        systemConfig.setResourceConfig(config);
    }
}

contract SystemConfig_Setters_Test is SystemConfig_Init {
    /// @dev Tests that `setBatcherHash` updates the batcher hash successfully.
    function testFuzz_setBatcherHash_succeeds(bytes32 newBatcherHash) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.BATCHER, abi.encode(newBatcherHash));

        vm.prank(systemConfig.owner());
        systemConfig.setBatcherHash(newBatcherHash);
        assertEq(systemConfig.batcherHash(), newBatcherHash);
    }

    /// @dev Tests that `setGasConfig` updates the overhead and scalar successfully.
    function testFuzz_setGasConfig_succeeds(uint256 newOverhead, uint256 newScalar) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.GAS_CONFIG, abi.encode(newOverhead, newScalar));

        vm.prank(systemConfig.owner());
        systemConfig.setGasConfig(newOverhead, newScalar);
        assertEq(systemConfig.overhead(), newOverhead);
        assertEq(systemConfig.scalar(), newScalar);
    }

    /// @dev Tests that `setGasLimit` updates the gas limit successfully.
    function testFuzz_setGasLimit_succeeds(uint64 newGasLimit) external {
        uint64 minimumGasLimit = systemConfig.minimumGasLimit();
        newGasLimit = uint64(bound(uint256(newGasLimit), uint256(minimumGasLimit), uint256(type(uint64).max)));

        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.GAS_LIMIT, abi.encode(newGasLimit));

        vm.prank(systemConfig.owner());
        systemConfig.setGasLimit(newGasLimit);
        assertEq(systemConfig.gasLimit(), newGasLimit);
    }

    /// @dev Tests that `setUnsafeBlockSigner` updates the block signer successfully.
    function testFuzz_setUnsafeBlockSigner_succeeds(address newUnsafeSigner) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SystemConfig.UpdateType.UNSAFE_BLOCK_SIGNER, abi.encode(newUnsafeSigner));

        vm.prank(systemConfig.owner());
        systemConfig.setUnsafeBlockSigner(newUnsafeSigner);
        assertEq(systemConfig.unsafeBlockSigner(), newUnsafeSigner);
    }
}
