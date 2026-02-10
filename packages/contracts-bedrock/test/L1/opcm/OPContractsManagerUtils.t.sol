// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { OPContractsManagerUtils } from "src/L1/opcm/OPContractsManagerUtils.sol";
import { OPContractsManagerContainer } from "src/L1/opcm/OPContractsManagerContainer.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Blueprint } from "src/libraries/Blueprint.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";

/// @title ImplV1_Harness
/// @notice Implementation contract with version 1.0.0 for testing upgrades.
contract OPContractsManagerUtils_ImplV1_Harness is ISemver {
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    function initialize() external { }
}

/// @title ImplV1b_Harness
/// @notice Another v1 implementation for testing same-version upgrades.
contract OPContractsManagerUtils_ImplV1b_Harness is ISemver {
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    function initialize() external { }
}

/// @title ImplV2_Harness
/// @notice Implementation contract with version 2.0.0 for testing upgrades.
contract OPContractsManagerUtils_ImplV2_Harness is ISemver {
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    function initialize() external { }
}

/// @title ImplV2Beta_Harness
/// @notice Implementation with beta prerelease tag for testing extra tag rejection.
contract OPContractsManagerUtils_ImplV2Beta_Harness is ISemver {
    /// @custom:semver 2.0.0-beta.1
    string public constant version = "2.0.0-beta.1";

    function initialize() external { }
}

/// @title ImplV2Interop_Harness
/// @notice Implementation with interop build metadata for testing extra tag rejection.
contract OPContractsManagerUtils_ImplV2Interop_Harness is ISemver {
    /// @custom:semver 2.0.0+interop
    string public constant version = "2.0.0+interop";

    function initialize() external { }
}

/// @title OPContractsManagerUtils_TestInit
/// @notice Shared setup for OPContractsManagerUtils tests.
contract OPContractsManagerUtils_TestInit is Test {
    OPContractsManagerUtils internal utils;
    OPContractsManagerContainer internal container;
    OPContractsManagerContainer.Blueprints internal blueprints;
    OPContractsManagerContainer.Implementations internal implementations;

    /// @notice Real StorageSetter used by utils.upgrade().
    IStorageSetter internal storageSetter;

    function setUp() public virtual {
        // Etch code into the magic testing address so we're recognized as a test env.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"01");

        // Deploy real StorageSetter using DeployUtils.
        storageSetter = IStorageSetter(
            DeployUtils.create1({
                _name: "StorageSetter",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IStorageSetter.__constructor__, ()))
            })
        );

        // Set up mock blueprints.
        blueprints = OPContractsManagerContainer.Blueprints({
            addressManager: makeAddr("addressManager"),
            proxy: makeAddr("proxy"),
            proxyAdmin: makeAddr("proxyAdmin"),
            l1ChugSplashProxy: makeAddr("l1ChugSplashProxy"),
            resolvedDelegateProxy: makeAddr("resolvedDelegateProxy")
        });

        // Set up implementations - use real StorageSetter, mocks for the rest.
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
            storageSetterImpl: address(storageSetter)
        });

        // Deploy the container and utils.
        container = new OPContractsManagerContainer(blueprints, implementations, bytes32(0));
        utils = new OPContractsManagerUtils(IOPContractsManagerContainer(address(container)));
    }

    /// @notice Helper to create an array of ExtraInstructions.
    /// @param _key The key of the instruction.
    /// @param _data The data of the instruction.
    /// @return The array of extra instructions.
    function _createInstructions(
        string memory _key,
        bytes memory _data
    )
        internal
        pure
        returns (OPContractsManagerUtils.ExtraInstruction[] memory)
    {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new OPContractsManagerUtils.ExtraInstruction[](1);
        instructions[0] = OPContractsManagerUtils.ExtraInstruction({ key: _key, data: _data });
        return instructions;
    }

    /// @notice Helper to create an empty array of ExtraInstructions.
    /// @return The empty array of extra instructions.
    function _emptyInstructions() internal pure returns (OPContractsManagerUtils.ExtraInstruction[] memory) {
        return new OPContractsManagerUtils.ExtraInstruction[](0);
    }
}

/// @title OPContractsManagerUtils_ChainIdToBatchInboxAddress_Test
/// @notice Tests the chainIdToBatchInboxAddress function.
contract OPContractsManagerUtils_ChainIdToBatchInboxAddress_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that chainIdToBatchInboxAddress produces deterministic, correctly formatted addresses.
    /// @param _chainId The chain ID to test.
    function testFuzz_chainIdToBatchInboxAddress_succeeds(uint256 _chainId) public view {
        address inbox = utils.chainIdToBatchInboxAddress(_chainId);

        // The version byte (first byte) should be 0x00.
        bytes20 inboxBytes = bytes20(inbox);
        assertEq(inboxBytes[0], 0x00, "First byte should be version byte 0x00");

        // Verify determinism by calling again.
        assertEq(utils.chainIdToBatchInboxAddress(_chainId), inbox, "Result should be deterministic");
    }

    /// @notice Tests that different chain IDs produce different batch inbox addresses.
    /// @param _chainId1 The first chain ID.
    /// @param _chainId2 The second chain ID.
    function testFuzz_chainIdToBatchInboxAddress_differentInputs_succeeds(
        uint256 _chainId1,
        uint256 _chainId2
    )
        public
        view
    {
        vm.assume(_chainId1 != _chainId2);

        address inbox1 = utils.chainIdToBatchInboxAddress(_chainId1);
        address inbox2 = utils.chainIdToBatchInboxAddress(_chainId2);

        assertNotEq(inbox1, inbox2, "Different chain IDs should produce different addresses");
    }
}

/// @title OPContractsManagerUtils_ComputeSalt_Test
/// @notice Tests the computeSalt function.
contract OPContractsManagerUtils_ComputeSalt_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that computeSalt produces deterministic output matching keccak256 encoding.
    /// @param _chainId The chain ID.
    /// @param _mixer The salt mixer.
    /// @param _name The contract name.
    function testFuzz_computeSalt_succeeds(
        uint256 _chainId,
        string calldata _mixer,
        string calldata _name
    )
        public
        view
    {
        bytes32 expected = keccak256(abi.encode(_chainId, _mixer, _name));
        bytes32 actual = utils.computeSalt(_chainId, _mixer, _name);

        assertEq(actual, expected, "Salt should match keccak256(abi.encode(...))");

        // Verify determinism by calling again.
        assertEq(utils.computeSalt(_chainId, _mixer, _name), actual, "Salt should be deterministic");
    }
}

/// @title OPContractsManagerUtils_HasInstruction_Test
/// @notice Tests the hasInstruction function.
contract OPContractsManagerUtils_HasInstruction_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that hasInstruction returns false for empty instructions array.
    function test_hasInstruction_emptyArray_succeeds() public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions = _emptyInstructions();

        assertFalse(utils.hasInstruction(instructions, "AnyKey", "AnyData"), "Empty array should return false");
    }

    /// @notice Tests that hasInstruction returns true when the instruction exists, false otherwise.
    /// @param _key The key to search for.
    /// @param _data The data to search for.
    function testFuzz_hasInstruction_exists_succeeds(string calldata _key, bytes calldata _data) public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new OPContractsManagerUtils.ExtraInstruction[](1);
        instructions[0] = OPContractsManagerUtils.ExtraInstruction({ key: _key, data: _data });

        assertTrue(utils.hasInstruction(instructions, _key, _data), "Should find matching instruction");
        assertFalse(utils.hasInstruction(instructions, "nonexistent", _data), "Wrong key returns false");
        assertFalse(utils.hasInstruction(instructions, _key, "nonexistent"), "Wrong data returns false");
    }

    /// @notice Tests hasInstruction finds correct instruction among multiple entries.
    function test_hasInstruction_multipleInstructions_succeeds() public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new OPContractsManagerUtils.ExtraInstruction[](3);
        instructions[0] = OPContractsManagerUtils.ExtraInstruction({ key: "Key1", data: bytes("Data1") });
        instructions[1] = OPContractsManagerUtils.ExtraInstruction({ key: "Key2", data: bytes("Data2") });
        instructions[2] = OPContractsManagerUtils.ExtraInstruction({ key: "Key3", data: bytes("Data3") });

        assertTrue(utils.hasInstruction(instructions, "Key1", "Data1"), "First instruction should be found");
        assertTrue(utils.hasInstruction(instructions, "Key2", "Data2"), "Second instruction should be found");
        assertTrue(utils.hasInstruction(instructions, "Key3", "Data3"), "Third instruction should be found");
        assertFalse(utils.hasInstruction(instructions, "Key4", "Data4"), "Non-existent should not be found");
    }
}

/// @title OPContractsManagerUtils_GetInstructionByKey_Test
/// @notice Tests the getInstructionByKey function.
contract OPContractsManagerUtils_GetInstructionByKey_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that getInstructionByKey returns empty for empty array.
    function test_getInstructionByKey_emptyArray_succeeds() public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions = _emptyInstructions();

        OPContractsManagerUtils.ExtraInstruction memory result = utils.getInstructionByKey(instructions, "AnyKey");

        assertEq(result.key, "", "Key should be empty");
        assertEq(result.data, bytes(""), "Data should be empty");
    }

    /// @notice Tests getInstructionByKey returns correct result when exists or empty when not.
    /// @param _key The key to search for.
    /// @param _data The data to associate with the key.
    function testFuzz_getInstructionByKey_succeeds(string calldata _key, bytes calldata _data) public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new OPContractsManagerUtils.ExtraInstruction[](1);
        instructions[0] = OPContractsManagerUtils.ExtraInstruction({ key: _key, data: _data });

        // Should find the instruction.
        OPContractsManagerUtils.ExtraInstruction memory found = utils.getInstructionByKey(instructions, _key);
        assertEq(found.key, _key, "Key should match");
        assertEq(found.data, _data, "Data should match");

        // Should not find a non-existent instruction.
        OPContractsManagerUtils.ExtraInstruction memory notFound =
            utils.getInstructionByKey(instructions, "nonexistent");
        assertEq(notFound.key, "", "Key should be empty for not found");
    }

    /// @notice Tests that getInstructionByKey returns the first matching instruction for dupes.
    function test_getInstructionByKey_duplicateKeys_succeeds() public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new OPContractsManagerUtils.ExtraInstruction[](2);
        instructions[0] = OPContractsManagerUtils.ExtraInstruction({ key: "DupeKey", data: bytes("FirstData") });
        instructions[1] = OPContractsManagerUtils.ExtraInstruction({ key: "DupeKey", data: bytes("SecondData") });

        OPContractsManagerUtils.ExtraInstruction memory result = utils.getInstructionByKey(instructions, "DupeKey");

        assertEq(result.data, bytes("FirstData"), "Should return first matching instruction");
    }
}

/// @title OPContractsManagerUtils_LoadBytes_Test
/// @notice Tests the loadBytes function.
contract OPContractsManagerUtils_LoadBytes_Test is OPContractsManagerUtils_TestInit {
    /// @notice Mock source contract for testing loadBytes.
    address internal mockSource;

    /// @notice Selector for the mock function.
    bytes4 internal constant MOCK_SELECTOR = bytes4(keccak256("getData()"));

    function setUp() public override {
        super.setUp();
        mockSource = makeAddr("mockSource");
    }

    /// @notice Tests that loadBytes returns data from the source when no override exists.
    function test_loadBytes_fromSource_succeeds() public {
        bytes memory expectedData = abi.encode("test data");

        // Mock the source to return expected data.
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), expectedData);

        bytes memory result = utils.loadBytes(mockSource, MOCK_SELECTOR, "testField", _emptyInstructions());

        assertEq(result, expectedData, "Should return data from source");
    }

    /// @notice Tests that loadBytes returns override data when an override instruction exists.
    /// @param _overrideData Fuzzed override data to test with.
    function testFuzz_loadBytes_withOverride_succeeds(bytes calldata _overrideData) public view {
        OPContractsManagerUtils.ExtraInstruction[] memory instructions = _createInstructions("testField", _overrideData);

        bytes memory result = utils.loadBytes(mockSource, MOCK_SELECTOR, "testField", instructions);

        assertEq(result, _overrideData, "Should return override data");
    }

    /// @notice Tests that loadBytes reverts when the source call fails.
    function test_loadBytes_sourceCallFails_reverts() public {
        // Mock the source to revert.
        vm.mockCallRevert(mockSource, abi.encodePacked(MOCK_SELECTOR), "source error");

        vm.expectRevert(
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_ConfigLoadFailed.selector, "testField"
            )
        );
        utils.loadBytes(mockSource, MOCK_SELECTOR, "testField", _emptyInstructions());
    }
}

/// @title OPContractsManagerUtils_LoadOrDeployProxy_Test
/// @notice Tests the loadOrDeployProxy function.
contract OPContractsManagerUtils_LoadOrDeployProxy_Test is OPContractsManagerUtils_TestInit {
    /// @notice Mock source contract for testing load behavior.
    address internal mockSource;

    /// @notice Real proxy admin for testing.
    IProxyAdmin internal proxyAdmin;

    /// @notice Real address manager for testing.
    IAddressManager internal addressManager;

    /// @notice Owner for ProxyAdmin.
    address internal owner;

    /// @notice Selector for the mock proxy getter.
    bytes4 internal constant MOCK_SELECTOR = bytes4(keccak256("getProxy()"));

    /// @notice ProxyDeployArgs for testing.
    OPContractsManagerUtils.ProxyDeployArgs internal deployArgs;

    function setUp() public override {
        super.setUp();

        owner = makeAddr("owner");
        mockSource = makeAddr("mockSource");

        // Deploy real ProxyAdmin.
        proxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (owner)))
            })
        );

        // Deploy real AddressManager.
        addressManager = IAddressManager(
            DeployUtils.create1({
                _name: "AddressManager",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IAddressManager.__constructor__, ()))
            })
        );

        // Transfer AddressManager ownership to ProxyAdmin.
        addressManager.transferOwnership(address(proxyAdmin));

        // Set AddressManager on ProxyAdmin.
        vm.prank(owner);
        proxyAdmin.setAddressManager(addressManager);

        deployArgs = OPContractsManagerUtils.ProxyDeployArgs({
            proxyAdmin: proxyAdmin,
            addressManager: addressManager,
            l2ChainId: 42,
            saltMixer: "testMixer"
        });
    }

    /// @notice Tests that loadOrDeployProxy returns the proxy from the source when it exists.
    /// @param _existingProxy Fuzzed address for the existing proxy.
    function testFuzz_loadOrDeployProxy_loadsExisting_succeeds(address _existingProxy) public {
        vm.assume(_existingProxy != address(0));

        // Mock the source to return the existing proxy.
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), abi.encode(_existingProxy));

        address result =
            utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", _emptyInstructions());

        assertEq(result, _existingProxy, "Should return existing proxy");
    }

    /// @notice Tests that loadOrDeployProxy reverts when load fails and deployment is not permitted.
    function test_loadOrDeployProxy_loadFailsNotPermitted_reverts() public {
        // Mock the source to revert.
        vm.mockCallRevert(mockSource, abi.encodePacked(MOCK_SELECTOR), "source error");

        vm.expectRevert(
            abi.encodeWithSelector(IOPContractsManagerUtils.OPContractsManagerUtils_ProxyMustLoad.selector, "TestProxy")
        );
        utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", _emptyInstructions());
    }

    /// @notice Tests that loadOrDeployProxy reverts when source returns zero address.
    function test_loadOrDeployProxy_zeroAddressNotPermitted_reverts() public {
        // Mock the source to return address(0).
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), abi.encode(address(0)));

        vm.expectRevert(
            abi.encodeWithSelector(IOPContractsManagerUtils.OPContractsManagerUtils_ProxyMustLoad.selector, "TestProxy")
        );
        utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", _emptyInstructions());
    }

    /// @notice Tests that specific contract permission bypasses ProxyMustLoad when load fails.
    function test_loadOrDeployProxy_specificPermission_succeeds() public {
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), abi.encode(address(0)));

        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            _createInstructions(Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, bytes("TestProxy"));

        // Permission check passes (no ProxyMustLoad error), but Blueprint deploy fails since mocked.
        vm.expectRevert(Blueprint.NotABlueprint.selector);
        utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", instructions);
    }

    /// @notice Tests that ALL permission bypasses ProxyMustLoad when load fails.
    function test_loadOrDeployProxy_allPermission_succeeds() public {
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), abi.encode(address(0)));

        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            _createInstructions(Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, Constants.PERMIT_ALL_CONTRACTS_INSTRUCTION);

        // Permission check passes (no ProxyMustLoad error), but Blueprint deploy fails since mocked.
        vm.expectRevert(Blueprint.NotABlueprint.selector);
        utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", instructions);
    }
}

/// @title OPContractsManagerUtils_Upgrade_Test
/// @notice Tests the upgrade function.
contract OPContractsManagerUtils_Upgrade_Test is OPContractsManagerUtils_TestInit {
    /// @notice Real proxy admin for testing (owned by utils).
    IProxyAdmin internal proxyAdmin;

    /// @notice Real proxy for testing.
    IProxy internal proxy;

    /// @notice v1 implementation for testing.
    OPContractsManagerUtils_ImplV1_Harness internal implV1;

    /// @notice Another v1 implementation for same-version testing.
    OPContractsManagerUtils_ImplV1b_Harness internal implV1b;

    /// @notice v2 implementation for testing.
    OPContractsManagerUtils_ImplV2_Harness internal implV2;

    /// @notice Storage slot to reset during upgrade (slot 0 for OZ Initializable).
    bytes32 internal constant TEST_SLOT = bytes32(uint256(0));

    /// @notice Byte offset within the slot for the initialized flag.
    uint8 internal constant TEST_OFFSET = 0;

    function setUp() public override {
        super.setUp();

        // Deploy real ProxyAdmin with utils as owner so utils.upgrade() can call proxyAdmin.
        proxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (address(utils))))
            })
        );

        // Deploy real Proxy with ProxyAdmin as admin.
        proxy = IProxy(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (address(proxyAdmin))))
            })
        );

        // Set proxy type on ProxyAdmin (utils is owner).
        vm.prank(address(utils));
        proxyAdmin.setProxyType(address(proxy), IProxyAdmin.ProxyType.ERC1967);

        // Deploy versioned implementations.
        implV1 = new OPContractsManagerUtils_ImplV1_Harness();
        implV1b = new OPContractsManagerUtils_ImplV1b_Harness();
        implV2 = new OPContractsManagerUtils_ImplV2_Harness();
    }

    /// @notice Tests that upgrade reverts when attempting a downgrade.
    function test_upgrade_downgradeNotAllowed_reverts() public {
        // Set v2 as current implementation.
        vm.prank(address(utils));
        proxyAdmin.upgrade(payable(address(proxy)), address(implV2));

        // Try to downgrade to v1 - should revert.
        vm.expectRevert(
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_DowngradeNotAllowed.selector, address(proxy)
            )
        );
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implV1),
            abi.encodeCall(OPContractsManagerUtils_ImplV1_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );
    }

    /// @notice Tests that upgrade allows upgrading to the same version.
    function test_upgrade_sameVersion_succeeds() public {
        // Set v1 as current implementation.
        vm.prank(address(utils));
        proxyAdmin.upgrade(payable(address(proxy)), address(implV1));

        // Upgrade to the same version (different contract) should succeed.
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implV1b),
            abi.encodeCall(OPContractsManagerUtils_ImplV1b_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );

        // Verify the implementation changed.
        assertEq(proxyAdmin.getProxyImplementation(payable(address(proxy))), address(implV1b));
    }

    /// @notice Tests that upgrade succeeds when upgrading to a newer version.
    function test_upgrade_newerVersion_succeeds() public {
        // Set v1 as current implementation.
        vm.prank(address(utils));
        proxyAdmin.upgrade(payable(address(proxy)), address(implV1));

        // Upgrade to v2 should succeed.
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implV2),
            abi.encodeCall(OPContractsManagerUtils_ImplV2_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );

        // Verify the implementation changed.
        assertEq(proxyAdmin.getProxyImplementation(payable(address(proxy))), address(implV2));
    }

    /// @notice Tests that upgrade succeeds when target has no implementation (fresh deploy).
    function test_upgrade_noExistingImplementation_succeeds() public {
        // Upgrade fresh proxy (no existing implementation) should succeed.
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implV1),
            abi.encodeCall(OPContractsManagerUtils_ImplV1_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );

        // Verify the implementation was set.
        assertEq(proxyAdmin.getProxyImplementation(payable(address(proxy))), address(implV1));
    }

    /// @notice Tests that upgrade reverts with prerelease tag in production environment.
    function test_upgrade_prereleaseInProd_reverts() public {
        // Remove testing environment marker to simulate production.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"");

        // Simulate mainnet.
        vm.chainId(1);

        OPContractsManagerUtils_ImplV2Beta_Harness implBeta = new OPContractsManagerUtils_ImplV2Beta_Harness();

        vm.expectRevert(
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_ExtraTagInProd.selector, address(implBeta)
            )
        );
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implBeta),
            abi.encodeCall(OPContractsManagerUtils_ImplV2Beta_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );
    }

    /// @notice Tests that upgrade reverts with build metadata tag in production environment.
    function test_upgrade_buildMetadataInProd_reverts() public {
        // Remove testing environment marker to simulate production.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"");

        // Simulate mainnet.
        vm.chainId(1);

        OPContractsManagerUtils_ImplV2Interop_Harness implInterop = new OPContractsManagerUtils_ImplV2Interop_Harness();

        vm.expectRevert(
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_ExtraTagInProd.selector, address(implInterop)
            )
        );
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implInterop),
            abi.encodeCall(OPContractsManagerUtils_ImplV2Interop_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );
    }

    /// @notice Tests that upgrade with extra tags succeeds when dev features are enabled.
    function test_upgrade_extraTagWithDevFeatures_succeeds() public {
        // Mock devFeatureBitmap to return non-zero (dev features enabled).
        vm.mockCall(
            address(container),
            abi.encodeCall(IOPContractsManagerContainer.devFeatureBitmap, ()),
            abi.encode(bytes32(uint256(1)))
        );

        OPContractsManagerUtils_ImplV2Beta_Harness implBeta = new OPContractsManagerUtils_ImplV2Beta_Harness();

        // Should succeed because dev features are enabled.
        utils.upgrade(
            proxyAdmin,
            address(proxy),
            address(implBeta),
            abi.encodeCall(OPContractsManagerUtils_ImplV2Beta_Harness.initialize, ()),
            TEST_SLOT,
            TEST_OFFSET
        );

        assertEq(proxyAdmin.getProxyImplementation(payable(address(proxy))), address(implBeta));
    }
}

/// @title OPContractsManagerUtils_Blueprints_Test
/// @notice Tests the blueprints() getter.
contract OPContractsManagerUtils_Blueprints_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that blueprints() returns the struct from the container.
    function test_blueprints_succeeds() public view {
        assertEq(abi.encode(utils.blueprints()), abi.encode(blueprints));
    }
}

/// @title OPContractsManagerUtils_Implementations_Test
/// @notice Tests the implementations() getter.
contract OPContractsManagerUtils_Implementations_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that implementations() returns the struct from the container.
    function test_implementations_succeeds() public view {
        assertEq(abi.encode(utils.implementations()), abi.encode(implementations));
    }
}

/// @title OPContractsManagerUtils_ContractsContainer_Test
/// @notice Tests the contractsContainer() getter.
contract OPContractsManagerUtils_ContractsContainer_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that contractsContainer() returns the container provided at construction.
    function test_contractsContainer_succeeds() public view {
        assertEq(address(utils.contractsContainer()), address(container));
    }
}

/// @title OPContractsManagerUtils_IsMatchingInstruction_Test
/// @notice Tests the isMatchingInstruction function.
contract OPContractsManagerUtils_IsMatchingInstruction_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that isMatchingInstruction returns true when the instruction matches the key and data.
    function testFuzz_isMatchingInstruction_succeeds(OPContractsManagerUtils.ExtraInstruction memory _instruction)
        public
        view
    {
        assertTrue(utils.isMatchingInstruction(_instruction, _instruction.key, _instruction.data));
    }

    /// @notice Tests that isMatchingInstruction returns false when the instruction does not match the key.
    function testFuzz_isMatchingInstruction_notMatchingKey_fails(
        OPContractsManagerUtils.ExtraInstruction memory _instruction
    )
        public
        view
    {
        // Create a key that is not the same as the instruction key.
        string memory _key = string.concat("not:", _instruction.key);

        assertFalse(utils.isMatchingInstruction(_instruction, _key, _instruction.data));
    }

    /// @notice Tests that isMatchingInstruction returns false when the instruction does not match the data.
    function testFuzz_isMatchingInstruction_notMatchingData_fails(
        OPContractsManagerUtils.ExtraInstruction memory _instruction
    )
        public
        view
    {
        // Create a data that is not the same as the instruction data.
        bytes memory _data = bytes.concat("not:", _instruction.data);

        assertFalse(utils.isMatchingInstruction(_instruction, _instruction.key, _data));
    }
}

/// @title OPContractsManagerUtils_IsMatchingInstructionByKey_Test
/// @notice Tests the isMatchingInstructionByKey function.
contract OPContractsManagerUtils_IsMatchingInstructionByKey_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that isMatchingInstructionByKey returns true when the instruction matches the key.
    function testFuzz_isMatchingInstructionByKey_succeeds(OPContractsManagerUtils.ExtraInstruction memory _instruction)
        public
        view
    {
        assertTrue(utils.isMatchingInstructionByKey(_instruction, _instruction.key));
    }

    /// @notice Tests that isMatchingInstructionKey returns false when the instruction does not match the key.
    function testFuzz_isMatchingInstructionByKey_notMatchingKey_fails(
        OPContractsManagerUtils.ExtraInstruction memory _instruction
    )
        public
        view
    {
        // Create a key that is not the same as the instruction key.
        string memory _key = string.concat("not:", _instruction.key);
        assertFalse(utils.isMatchingInstructionByKey(_instruction, _key));
    }
}
