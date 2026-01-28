// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { OPContractsManagerContainer } from "src/L1/opcm/OPContractsManagerContainer.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

/// @title OPContractsManagerContainer_TestInit
/// @notice Shared setup for OPContractsManagerContainer tests.
contract OPContractsManagerContainer_TestInit is Test {
    OPContractsManagerContainer.Blueprints internal blueprints;
    OPContractsManagerContainer.Implementations internal implementations;

    function setUp() public virtual {
        blueprints = OPContractsManagerContainer.Blueprints({
            addressManager: makeAddr("addressManager"),
            proxy: makeAddr("proxy"),
            proxyAdmin: makeAddr("proxyAdmin"),
            l1ChugSplashProxy: makeAddr("l1ChugSplashProxy"),
            resolvedDelegateProxy: makeAddr("resolvedDelegateProxy")
        });

        implementations = OPContractsManagerContainer.Implementations({
            superchainConfigImpl: makeAddr("superchainConfigImpl"),
            protocolVersionsImpl: makeAddr("protocolVersionsImpl"),
            l1ERC721BridgeImpl: makeAddr("l1ERC721BridgeImpl"),
            optimismPortalImpl: makeAddr("optimismPortalImpl"),
            optimismPortalInteropImpl: makeAddr("optimismPortalInteropImpl"),
            ethLockboxImpl: makeAddr("ethLockboxImpl"),
            systemConfigImpl: makeAddr("systemConfigImpl"),
            optimismMintableERC20FactoryImpl: makeAddr("optimismMintableERC20FactoryImpl"),
            l1CrossDomainMessengerImpl: makeAddr("l1CrossDomainMessengerImpl"),
            l1StandardBridgeImpl: makeAddr("l1StandardBridgeImpl"),
            disputeGameFactoryImpl: makeAddr("disputeGameFactoryImpl"),
            anchorStateRegistryImpl: makeAddr("anchorStateRegistryImpl"),
            delayedWETHImpl: makeAddr("delayedWETHImpl"),
            mipsImpl: makeAddr("mipsImpl"),
            faultDisputeGameImpl: makeAddr("faultDisputeGameImpl"),
            permissionedDisputeGameImpl: makeAddr("permissionedDisputeGameImpl"),
            superFaultDisputeGameImpl: makeAddr("superFaultDisputeGameImpl"),
            superPermissionedDisputeGameImpl: makeAddr("superPermissionedDisputeGameImpl"),
            storageSetterImpl: makeAddr("storageSetterImpl")
        });
    }

    /// @notice Deploys a new OPContractsManagerContainer with the given dev feature bitmap.
    /// @param _devFeatureBitmap The dev feature bitmap to use.
    /// @return The deployed OPContractsManagerContainer.
    function _deploy(bytes32 _devFeatureBitmap) internal returns (OPContractsManagerContainer) {
        return new OPContractsManagerContainer(blueprints, implementations, _devFeatureBitmap);
    }
}

/// @title OPContractsManagerContainer_Constructor_Test
/// @notice Tests the constructor of OPContractsManagerContainer.
contract OPContractsManagerContainer_Constructor_Test is OPContractsManagerContainer_TestInit {
    /// @notice Tests that the constructor succeeds with any dev bitmap when in a test environment.
    /// @param _chainId The chain ID to use.
    /// @param _devFeatureBitmap The dev feature bitmap to use.
    function testFuzz_constructor_devBitmapInTestEnv_succeeds(uint64 _chainId, bytes32 _devFeatureBitmap) public {
        // Etch code into the magic testing address so we're recognized as a test env.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"01");

        // Set chain ID.
        vm.chainId(_chainId);

        OPContractsManagerContainer container = _deploy(_devFeatureBitmap);

        assertEq(container.devFeatureBitmap(), _devFeatureBitmap);
    }

    /// @notice Tests that the constructor reverts when dev features are enabled on mainnet without
    ///         test env.
    /// @param _devFeatureBitmap The dev feature bitmap to use.
    function testFuzz_constructor_devBitmapOnMainnet_reverts(bytes32 _devFeatureBitmap) public {
        // Ensure at least one dev feature is enabled.
        _devFeatureBitmap = bytes32(bound(uint256(_devFeatureBitmap), 1, type(uint256).max));

        // Clear the magic testing address so we're recognized as production.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"");

        // Set chain ID to mainnet.
        vm.chainId(1);

        vm.expectRevert(OPContractsManagerContainer.OPContractsManagerContainer_DevFeatureInProd.selector);
        _deploy(_devFeatureBitmap);
    }

    /// @notice Tests that the constructor succeeds on mainnet with a zero dev bitmap.
    function test_constructor_zeroBitmapOnMainnet_succeeds() public {
        // Clear the magic testing address.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"");

        // Set chain ID to mainnet.
        vm.chainId(1);

        OPContractsManagerContainer container = _deploy(bytes32(0));

        assertEq(container.devFeatureBitmap(), bytes32(0));
    }
}

/// @title OPContractsManagerContainer_Blueprints_Test
/// @notice Tests the blueprints() getter.
contract OPContractsManagerContainer_Blueprints_Test is OPContractsManagerContainer_TestInit {
    /// @notice Tests that blueprints() returns the struct provided at construction.
    function test_blueprints_succeeds() public {
        OPContractsManagerContainer container = _deploy(bytes32(0));

        assertEq(abi.encode(container.blueprints()), abi.encode(blueprints));
    }
}

/// @title OPContractsManagerContainer_Implementations_Test
/// @notice Tests the implementations() getter.
contract OPContractsManagerContainer_Implementations_Test is OPContractsManagerContainer_TestInit {
    /// @notice Tests that implementations() returns the struct provided at construction.
    function test_implementations_succeeds() public {
        OPContractsManagerContainer container = _deploy(bytes32(0));

        assertEq(abi.encode(container.implementations()), abi.encode(implementations));
    }
}

/// @title OPContractsManagerContainer_IsDevFeatureEnabled_Test
/// @notice Tests the isDevFeatureEnabled() function.
contract OPContractsManagerContainer_IsDevFeatureEnabled_Test is OPContractsManagerContainer_TestInit {
    /// @notice Tests that isDevFeatureEnabled returns true when the feature bit is set.
    /// @param _bitIndex The bit index to test.
    function testFuzz_isDevFeatureEnabled_bitSet_succeeds(uint8 _bitIndex) public {
        bytes32 bitmap = bytes32(uint256(1) << _bitIndex);
        bytes32 feature = bytes32(uint256(1) << _bitIndex);

        OPContractsManagerContainer container = _deploy(bitmap);

        assertTrue(container.isDevFeatureEnabled(feature));
        assertFalse(container.isDevFeatureEnabled(bytes32(0)));
    }

    /// @notice Tests that isDevFeatureEnabled returns false when the feature bit is not set.
    /// @param _bitIndex The bit index to test.
    function testFuzz_isDevFeatureEnabled_bitNotSet_succeeds(uint8 _bitIndex) public {
        // Create a bitmap with all bits set except the one we're testing.
        bytes32 bitmap = bytes32(type(uint256).max ^ (uint256(1) << _bitIndex));
        bytes32 feature = bytes32(uint256(1) << _bitIndex);

        OPContractsManagerContainer container = _deploy(bitmap);

        assertFalse(container.isDevFeatureEnabled(feature));
    }

    /// @notice Tests that isDevFeatureEnabled returns false when the bitmap is zero.
    /// @param _feature The feature to check.
    function testFuzz_isDevFeatureEnabled_zeroBitmap_succeeds(bytes32 _feature) public {
        OPContractsManagerContainer container = _deploy(bytes32(0));

        assertFalse(container.isDevFeatureEnabled(_feature));
    }

    /// @notice Tests that isDevFeatureEnabled returns true for multiple features set at once.
    function test_isDevFeatureEnabled_multipleBitsSet_succeeds() public {
        uint256 numFeatures = vm.randomUint(1, 16);
        uint256 bitmap;
        uint8[] memory bitIndices = new uint8[](numFeatures);

        // Set random bits in the bitmap.
        for (uint256 i = 0; i < numFeatures; i++) {
            uint8 bitIndex = uint8(vm.randomUint(0, 255));
            bitIndices[i] = bitIndex;
            bitmap |= uint256(1) << bitIndex;
        }

        OPContractsManagerContainer container = _deploy(bytes32(bitmap));

        // Verify each feature is enabled.
        for (uint256 i = 0; i < numFeatures; i++) {
            bytes32 feature = bytes32(uint256(1) << bitIndices[i]);
            assertTrue(container.isDevFeatureEnabled(feature));
        }
    }
}

/// @title OPContractsManagerContainer_DevFeatureBitmap_Test
/// @notice Tests the devFeatureBitmap() getter.
contract OPContractsManagerContainer_DevFeatureBitmap_Test is OPContractsManagerContainer_TestInit {
    /// @notice Tests that devFeatureBitmap() returns the value provided at construction.
    /// @param _devFeatureBitmap The dev feature bitmap to use.
    function testFuzz_devFeatureBitmap_succeeds(bytes32 _devFeatureBitmap) public {
        OPContractsManagerContainer container = _deploy(_devFeatureBitmap);

        assertEq(container.devFeatureBitmap(), _devFeatureBitmap);
    }
}
