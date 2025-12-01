// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";

// Contracts
import { OPContractsManagerUtils } from "src/L1/opcm/OPContractsManagerUtils.sol";
import { OPContractsManagerContainer } from "src/L1/opcm/OPContractsManagerContainer.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Blueprint } from "src/libraries/Blueprint.sol";

// Interfaces
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";

/// @title OPContractsManagerUtils_TestInit
/// @notice Shared setup for OPContractsManagerUtils tests.
contract OPContractsManagerUtils_TestInit is Test {
    OPContractsManagerUtils internal utils;
    OPContractsManagerContainer internal container;
    OPContractsManagerContainer.Blueprints internal blueprints;
    OPContractsManagerContainer.Implementations internal implementations;

    function setUp() public virtual {
        // Etch code into the magic testing address so we're recognized as a test env.
        vm.etch(Constants.TESTING_ENVIRONMENT_ADDRESS, hex"01");

        // Set up mock blueprints.
        blueprints = OPContractsManagerContainer.Blueprints({
            addressManager: makeAddr("addressManager"),
            proxy: makeAddr("proxy"),
            proxyAdmin: makeAddr("proxyAdmin"),
            l1ChugSplashProxy: makeAddr("l1ChugSplashProxy"),
            resolvedDelegateProxy: makeAddr("resolvedDelegateProxy"),
            permissionedDisputeGame1: makeAddr("permissionedDisputeGame1"),
            permissionedDisputeGame2: makeAddr("permissionedDisputeGame2"),
            permissionlessDisputeGame1: makeAddr("permissionlessDisputeGame1"),
            permissionlessDisputeGame2: makeAddr("permissionlessDisputeGame2")
        });

        // Set up mock implementations.
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
            faultDisputeGameV2Impl: makeAddr("faultDisputeGameV2Impl"),
            permissionedDisputeGameV2Impl: makeAddr("permissionedDisputeGameV2Impl"),
            superFaultDisputeGameImpl: makeAddr("superFaultDisputeGameImpl"),
            superPermissionedDisputeGameImpl: makeAddr("superPermissionedDisputeGameImpl"),
            storageSetterImpl: makeAddr("storageSetterImpl")
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
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            _createInstructions("testField", _overrideData);

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
    /// @notice Mock source contract for testing loadOrDeployProxy.
    address internal mockSource;

    /// @notice Mock proxy admin for testing.
    IProxyAdmin internal mockProxyAdmin;

    /// @notice Mock address manager for testing.
    IAddressManager internal mockAddressManager;

    /// @notice Selector for the mock proxy getter.
    bytes4 internal constant MOCK_SELECTOR = bytes4(keccak256("getProxy()"));

    /// @notice ProxyDeployArgs for testing.
    OPContractsManagerUtils.ProxyDeployArgs internal deployArgs;

    function setUp() public override {
        super.setUp();
        mockSource = makeAddr("mockSource");
        mockProxyAdmin = IProxyAdmin(makeAddr("mockProxyAdmin"));
        mockAddressManager = IAddressManager(makeAddr("mockAddressManager"));

        deployArgs = OPContractsManagerUtils.ProxyDeployArgs({
            proxyAdmin: mockProxyAdmin,
            addressManager: mockAddressManager,
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

    /// @notice Tests that specific contract permission allows deployment when load fails.
    function test_loadOrDeployProxy_specificPermission_succeeds() public {
        // Mock the source to return address(0).
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), abi.encode(address(0)));

        // Create instruction permitting deployment of this specific contract.
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            _createInstructions(Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, bytes("TestProxy"));

        // Should not revert - but will revert on blueprint deployment since blueprints are mocks.
        // We verify the permission check passed by checking the error is from Blueprint, not ProxyMustLoad.
        vm.expectRevert(Blueprint.NotABlueprint.selector);
        utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", instructions);
    }

    /// @notice Tests that ALL permission allows deployment when load fails.
    function test_loadOrDeployProxy_allPermission_succeeds() public {
        // Mock the source to return address(0).
        vm.mockCall(mockSource, abi.encodePacked(MOCK_SELECTOR), abi.encode(address(0)));

        // Create instruction permitting deployment of all contracts.
        OPContractsManagerUtils.ExtraInstruction[] memory instructions =
            _createInstructions(Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, Constants.PERMIT_ALL_CONTRACTS_INSTRUCTION);

        // Should not revert with ProxyMustLoad - will fail on Blueprint since blueprints are mocks.
        vm.expectRevert(Blueprint.NotABlueprint.selector);
        utils.loadOrDeployProxy(mockSource, MOCK_SELECTOR, deployArgs, "TestProxy", instructions);
    }
}

/// @title OPContractsManagerUtils_Upgrade_Test
/// @notice Tests the upgrade function.
contract OPContractsManagerUtils_Upgrade_Test is OPContractsManagerUtils_TestInit {
    /// @notice Mock proxy admin for testing.
    address internal mockProxyAdmin;

    /// @notice Mock target proxy for testing.
    address internal mockTarget;

    /// @notice Mock implementation for testing.
    address internal mockImplementation;

    /// @notice Test storage slot.
    bytes32 internal constant TEST_SLOT = bytes32(uint256(0));

    /// @notice Test offset.
    uint8 internal constant TEST_OFFSET = 0;

    function setUp() public override {
        super.setUp();
        mockProxyAdmin = makeAddr("mockProxyAdmin");
        mockTarget = makeAddr("mockTarget");
        mockImplementation = makeAddr("mockImplementation");
    }

    /// @notice Tests that upgrade reverts when attempting a downgrade.
    function test_upgrade_downgradeNotAllowed_reverts() public {
        // Mock the proxy admin to return the current implementation.
        vm.mockCall(
            mockProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (payable(mockTarget))),
            abi.encode(mockImplementation)
        );

        // Mock the target to return a newer version (simulating a downgrade attempt).
        vm.mockCall(mockTarget, abi.encodeCall(ISemver.version, ()), abi.encode("2.0.0"));

        // Mock the new implementation to return an older version.
        vm.mockCall(mockImplementation, abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));

        vm.expectRevert(
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_DowngradeNotAllowed.selector, mockTarget
            )
        );
        utils.upgrade(IProxyAdmin(mockProxyAdmin), mockTarget, mockImplementation, "", TEST_SLOT, TEST_OFFSET);
    }

    /// @notice Tests that upgrade allows upgrading to the same version.
    function test_upgrade_sameVersion_succeeds() public {
        // Mock the proxy admin to return the current implementation.
        vm.mockCall(
            mockProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (payable(mockTarget))),
            abi.encode(mockImplementation)
        );

        // Both target and implementation return the same version.
        vm.mockCall(mockTarget, abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        vm.mockCall(mockImplementation, abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));

        // Mock the storage setter and proxy admin calls.
        _mockUpgradeCalls();

        // Should not revert.
        utils.upgrade(IProxyAdmin(mockProxyAdmin), mockTarget, mockImplementation, "", TEST_SLOT, TEST_OFFSET);
    }

    /// @notice Tests that upgrade succeeds when upgrading to a newer version.
    function test_upgrade_newerVersion_succeeds() public {
        // Mock the proxy admin to return the current implementation.
        vm.mockCall(
            mockProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (payable(mockTarget))),
            abi.encode(mockImplementation)
        );

        // Target is older, implementation is newer.
        vm.mockCall(mockTarget, abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        vm.mockCall(mockImplementation, abi.encodeCall(ISemver.version, ()), abi.encode("2.0.0"));

        // Mock the storage setter and proxy admin calls.
        _mockUpgradeCalls();

        // Should not revert.
        utils.upgrade(IProxyAdmin(mockProxyAdmin), mockTarget, mockImplementation, "", TEST_SLOT, TEST_OFFSET);
    }

    /// @notice Tests that upgrade succeeds when target has no implementation (fresh deploy).
    function test_upgrade_noExistingImplementation_succeeds() public {
        // Mock the proxy admin to return address(0) (no existing implementation).
        vm.mockCall(
            mockProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (payable(mockTarget))),
            abi.encode(address(0))
        );

        // Mock the storage setter and proxy admin calls.
        _mockUpgradeCalls();

        // Should not revert.
        utils.upgrade(IProxyAdmin(mockProxyAdmin), mockTarget, mockImplementation, "", TEST_SLOT, TEST_OFFSET);
    }

    /// @notice Helper to mock the upgrade calls.
    function _mockUpgradeCalls() internal {
        // Mock upgrade to storage setter.
        vm.mockCall(
            mockProxyAdmin,
            abi.encodeCall(IProxyAdmin.upgrade, (payable(mockTarget), implementations.storageSetterImpl)),
            ""
        );

        // Mock getBytes32 on target (via StorageSetter).
        vm.mockCall(mockTarget, abi.encodeCall(IStorageSetter.getBytes32, (TEST_SLOT)), abi.encode(bytes32(0)));

        // Mock setBytes32 on target (via StorageSetter).
        // Using abi.encodeWithSignature to disambiguate the overloaded function.
        vm.mockCall(mockTarget, abi.encodeWithSignature("setBytes32(bytes32,bytes32)", TEST_SLOT, bytes32(0)), "");

        // Mock upgradeAndCall.
        vm.mockCall(
            mockProxyAdmin,
            abi.encodeCall(IProxyAdmin.upgradeAndCall, (payable(mockTarget), mockImplementation, "")),
            ""
        );
    }
}

/// @title OPContractsManagerUtils_Constructor_Test
/// @notice Tests the constructor of OPContractsManagerUtils.
contract OPContractsManagerUtils_Constructor_Test is OPContractsManagerUtils_TestInit {
    /// @notice Tests that the constructor sets the contractsContainer correctly.
    function test_constructor_succeeds() public view {
        assertEq(address(utils.contractsContainer()), address(container));
    }
}
