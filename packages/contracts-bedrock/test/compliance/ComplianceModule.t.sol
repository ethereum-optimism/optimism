// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { stdStorage, StdStorage } from "forge-std/StdStorage.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { OptimismPortal2_TestInit } from "test/setup/OptimismPortal2TestInit.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";

// Contracts
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";

/// @title ComplianceModule_TestInit
/// @notice Sets up the Portal's bridge hook with separate owners for proxy upgrades and
///         policy changes.
abstract contract ComplianceModule_TestInit is CommonTest {
    event Held(bytes32 indexed id, BridgeHookTypes.Item item);
    event VerdictRecorded(bytes32 indexed id, uint64 expiresAt);
    event VerdictRevoked(bytes32 indexed id);
    event Completed(bytes32 indexed id);
    event PolicySet(IPolicy indexed previousPolicy, IPolicy indexed newPolicy);
    event Disabled(uint256 cutoffL2Nonce);
    event Enabled();

    address internal custodian = makeAddr("custodian");
    address internal moduleOwner = makeAddr("moduleOwner");
    IPolicy internal policy = IPolicy(makeAddr("policy"));

    IProxyAdmin internal moduleProxyAdmin;
    IComplianceModule internal impl;
    IComplianceModule internal module;

    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.BRIDGE_HOOK);

        moduleProxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (custodian)))
            })
        );
        impl = IComplianceModule(
            DeployUtils.create1({
                _name: "ComplianceModule",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IComplianceModule.__constructor__, ()))
            })
        );

        module = _newProxy();
        vm.prank(custodian);
        module.initialize(moduleOwner, policy, optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);

        vm.store(
            address(optimismPortal2),
            bytes32(ForgeArtifacts.getSlot("OptimismPortal2", "bridgeHook").slot),
            bytes32(uint256(uint160(address(module))))
        );
    }

    /// @notice Creates a proxy with the module implementation installed, but leaves module state
    ///         uninitialized so tests can exercise initialization.
    function _newProxy() internal returns (IComplianceModule proxy_) {
        proxy_ = IComplianceModule(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IProxy.__constructor__, (address(moduleProxyAdmin)))
                )
            })
        );
        vm.prank(custodian);
        moduleProxyAdmin.upgrade(payable(address(proxy_)), address(impl));
    }

    /// @notice Writes a module record directly, which hides the storage slot math.
    function _storeRecord(bytes32 _id, uint64 _heldAt, uint64 _expiresAt) internal {
        bytes32 slot = keccak256(abi.encode(_id, ForgeArtifacts.getSlot("ComplianceModule", "_items").slot));
        vm.store(address(module), slot, bytes32(uint256(_heldAt) | uint256(_expiresAt) << 64));
    }
}

/// @title ComplianceModule_Constructor_Test
/// @notice Tests the constructor of the `ComplianceModule` contract.
contract ComplianceModule_Constructor_Test is ComplianceModule_TestInit {
    /// @notice Tests that the implementation rejects initialization.
    function test_constructor_succeeds() external {
        vm.expectRevert(IComplianceModule.InvalidInitialization.selector);
        vm.prank(custodian);
        impl.initialize(moduleOwner, policy, optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
    }
}

/// @title ComplianceModule_Initialize_Test
/// @notice Tests the `initialize` function of the `ComplianceModule` contract.
contract ComplianceModule_Initialize_Test is ComplianceModule_TestInit {
    /// @notice Tests that initialize sets the owner, policy and addresses and emits both events.
    function test_initialize_succeeds() external {
        IComplianceModule proxy = _newProxy();

        vm.expectEmit(address(proxy));
        emit OwnershipTransferred(address(0), moduleOwner);
        vm.expectEmit(address(proxy));
        emit PolicySet(IPolicy(address(0)), policy);

        vm.prank(custodian);
        proxy.initialize(moduleOwner, policy, optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);

        assertEq(proxy.owner(), moduleOwner);
        assertEq(address(proxy.policy()), address(policy));
        assertEq(address(proxy.portal()), address(optimismPortal2));
        assertEq(address(proxy.l1CrossDomainMessenger()), address(l1CrossDomainMessenger));
        assertEq(address(proxy.bridge()), address(l1StandardBridge));
    }

    /// @notice Tests that initialize reverts for a caller other than the ProxyAdmin or its owner.
    function testFuzz_initialize_notProxyAdminOrOwner_reverts(address _caller) external {
        vm.assume(_caller != address(moduleProxyAdmin) && _caller != custodian);
        IComplianceModule proxy = _newProxy();

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        vm.prank(_caller);
        proxy.initialize(moduleOwner, policy, optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
    }

    /// @notice Tests that a second initialize reverts.
    function test_initialize_alreadyInitialized_reverts() external {
        vm.expectRevert(IComplianceModule.InvalidInitialization.selector);
        vm.prank(custodian);
        module.initialize(moduleOwner, policy, optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
    }

    /// @notice Tests that initialize reverts for a zero owner.
    function test_initialize_zeroOwner_reverts() external {
        IComplianceModule proxy = _newProxy();

        vm.expectRevert(IComplianceModule.ComplianceModule_ZeroAddress.selector);
        vm.prank(custodian);
        proxy.initialize(address(0), policy, optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
    }

    /// @notice Tests that initialize reverts for a zero policy.
    function test_initialize_zeroPolicy_reverts() external {
        IComplianceModule proxy = _newProxy();

        vm.expectRevert(IComplianceModule.ComplianceModule_ZeroAddress.selector);
        vm.prank(custodian);
        proxy.initialize(moduleOwner, IPolicy(address(0)), optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
    }

    /// @notice Tests that initialize reverts when the Portal, messenger or bridge is zero.
    function test_initialize_zeroAddress_reverts() external {
        address[3][3] memory args = [
            [address(0), address(l1CrossDomainMessenger), address(l1StandardBridge)],
            [address(optimismPortal2), address(0), address(l1StandardBridge)],
            [address(optimismPortal2), address(l1CrossDomainMessenger), address(0)]
        ];
        for (uint256 i; i < args.length; i++) {
            IComplianceModule proxy = _newProxy();
            vm.expectRevert(IComplianceModule.ComplianceModule_ZeroAddress.selector);
            vm.prank(custodian);
            proxy.initialize(
                moduleOwner,
                policy,
                IOptimismPortal2(payable(args[i][0])),
                IL1CrossDomainMessenger(args[i][1]),
                IL1StandardBridge(payable(args[i][2]))
            );
        }
    }
}

/// @title ComplianceModule_RenounceOwnership_Test
/// @notice Tests the `renounceOwnership` function of the `ComplianceModule` contract.
contract ComplianceModule_RenounceOwnership_Test is ComplianceModule_TestInit {
    /// @notice Tests that renounceOwnership reverts, even for the owner.
    function test_renounceOwnership_owner_reverts() external {
        vm.expectRevert(IComplianceModule.ComplianceModule_OwnershipNotRenounceable.selector);
        vm.prank(moduleOwner);
        module.renounceOwnership();
    }
}

/// @title ComplianceModule_TransferOwnership_Test
/// @notice Tests the `transferOwnership` function of the `ComplianceModule` contract.
contract ComplianceModule_TransferOwnership_Test is ComplianceModule_TestInit {
    /// @notice Tests that the owner does not change until the new owner accepts.
    function test_transferOwnership_succeeds() external {
        address newOwner = makeAddr("newOwner");

        vm.prank(moduleOwner);
        module.transferOwnership(newOwner);
        assertEq(module.owner(), moduleOwner);
        assertEq(module.pendingOwner(), newOwner);

        vm.prank(newOwner);
        module.acceptOwnership();
        assertEq(module.owner(), newOwner);
        assertEq(module.pendingOwner(), address(0));
    }
}

/// @title ComplianceModule_Paused_Test
/// @notice Tests the `paused` function of the `ComplianceModule` contract.
contract ComplianceModule_Paused_Test is ComplianceModule_TestInit {
    /// @notice Tests that paused returns the Portal's pause state.
    function test_paused_succeeds() external {
        assertFalse(module.paused());

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        assertTrue(module.paused());
    }
}

/// @title ComplianceModule_Parties_Test
/// @notice Tests the `parties` function of the `ComplianceModule` contract.
contract ComplianceModule_Parties_Test is ComplianceModule_TestInit {
    /// @notice Tests that parties and deposit screening use the bridge transfer's sender and recipient.
    function test_parties_bridgeEnvelope_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });

        vm.mockCall(address(policy), abi.encodeCall(IPolicy.screenDeposit, (item, parties)), abi.encode(true));
        vm.expectCall(address(policy), abi.encodeCall(IPolicy.screenDeposit, (item, parties)), 1);
        vm.prank(address(optimismPortal2));
        module.screenDeposit(item);

        IPolicy.Parties memory resolved = module.parties(item);
        assertEq(resolved.sender, alice);
        assertEq(resolved.recipient, bob);
    }

    /// @notice Tests that parties and screenDeposit use the unaliased sender and target.
    function test_parties_aliasedSender_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: AddressAliasHelper.applyL1ToL2Alias(alice),
            aliased: true,
            to: bob,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });

        vm.mockCall(address(policy), abi.encodeCall(IPolicy.screenDeposit, (item, parties)), abi.encode(true));
        vm.expectCall(address(policy), abi.encodeCall(IPolicy.screenDeposit, (item, parties)), 1);
        vm.prank(address(optimismPortal2));
        module.screenDeposit(item);

        IPolicy.Parties memory resolved = module.parties(item);
        assertEq(resolved.sender, alice);
        assertEq(resolved.recipient, bob);
    }
}

/// @title ComplianceModule_ScreenDeposit_Test
/// @notice Tests the `screenDeposit` function of the `ComplianceModule` contract.
contract ComplianceModule_ScreenDeposit_Test is ComplianceModule_TestInit {
    /// @notice Tests that screenDeposit reverts for a caller other than the Portal.
    function testFuzz_screenDeposit_notPortal_reverts(address _caller) external {
        vm.assume(_caller != address(optimismPortal2));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPortal.selector);
        vm.prank(_caller);
        module.screenDeposit(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Deposit,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 1 ether,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that screenDeposit reverts when the module is not the Portal's bridge hook.
    function test_screenDeposit_notActiveHook_reverts() external {
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.bridgeHook, ()), abi.encode(makeAddr("otherHook"))
        );

        vm.expectRevert(IComplianceModule.ComplianceModule_NotActiveHook.selector);
        vm.prank(address(optimismPortal2));
        module.screenDeposit(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Deposit,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 1 ether,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that screenDeposit passes the item and parties to the policy and returns its result.
    function testFuzz_screenDeposit_policyAnswer_succeeds(bool _pass) external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });

        vm.mockCall(address(policy), abi.encodeCall(IPolicy.screenDeposit, (item, parties)), abi.encode(_pass));
        vm.expectCall(address(policy), abi.encodeCall(IPolicy.screenDeposit, (item, parties)), 1);
        vm.prank(address(optimismPortal2));
        assertEq(module.screenDeposit(item), _pass);
    }

    /// @notice Tests that policy revert data is preserved.
    function test_screenDeposit_policyReverts_reverts() external {
        vm.mockCallRevert(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), hex"c0ffee");

        vm.expectRevert(bytes(hex"c0ffee"));
        vm.prank(address(optimismPortal2));
        module.screenDeposit(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Deposit,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 1 ether,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that screenDeposit returns true without calling the policy while disabled.
    function test_screenDeposit_disabled_succeeds() external {
        vm.prank(moduleOwner);
        module.disable(0);

        vm.expectCall(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), 0);
        vm.prank(address(optimismPortal2));
        assertTrue(
            module.screenDeposit(
                BridgeHookTypes.Item({
                    direction: BridgeHookTypes.Direction.Deposit,
                    asset: BridgeHookTypes.Asset.ETH,
                    from: alice,
                    aliased: false,
                    to: bob,
                    ethAmount: 1 ether,
                    l2Value: 0,
                    gasLimit: 100_000,
                    isCreation: false,
                    data: hex"",
                    nonce: 1
                })
            )
        );
    }
}

/// @title ComplianceModule_ScreenWithdrawal_Test
/// @notice Tests the `screenWithdrawal` function of the `ComplianceModule` contract.
contract ComplianceModule_ScreenWithdrawal_Test is ComplianceModule_TestInit {
    /// @notice Tests that screenWithdrawal reverts for a caller other than the Portal.
    function testFuzz_screenWithdrawal_notPortal_reverts(address _caller) external {
        vm.assume(_caller != address(optimismPortal2));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPortal.selector);
        vm.prank(_caller);
        module.screenWithdrawal(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 1 ether,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            }),
            0
        );
    }

    /// @notice Tests that screenWithdrawal reverts when the module is not the Portal's bridge hook.
    function test_screenWithdrawal_notActiveHook_reverts() external {
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.bridgeHook, ()), abi.encode(makeAddr("otherHook"))
        );

        vm.expectRevert(IComplianceModule.ComplianceModule_NotActiveHook.selector);
        vm.prank(address(optimismPortal2));
        module.screenWithdrawal(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 1 ether,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            }),
            0
        );
    }

    /// @notice Tests that screenWithdrawal passes its inputs to the policy and returns the result.
    function testFuzz_screenWithdrawal_policyAnswer_succeeds(
        bool _pass,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        external
    {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });
        _storeRecord(BridgeHookHashing.hashItem(item), 0, _expiresAt);

        bytes memory policyCall = abi.encodeCall(IPolicy.screenWithdrawal, (item, parties, _expiresAt, _finalizableAt));
        vm.mockCall(address(policy), policyCall, abi.encode(_pass));
        vm.expectCall(address(policy), policyCall, 1);
        vm.prank(address(optimismPortal2));
        assertEq(module.screenWithdrawal(item, _finalizableAt), _pass);
    }

    /// @notice Tests that policy revert data is preserved.
    function test_screenWithdrawal_policyReverts_reverts() external {
        vm.mockCallRevert(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), hex"c0ffee");

        vm.expectRevert(bytes(hex"c0ffee"));
        vm.prank(address(optimismPortal2));
        module.screenWithdrawal(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 1 ether,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            }),
            0
        );
    }

    /// @notice Tests that screenWithdrawal skips the policy when disabled at the cutoff nonce.
    function testFuzz_screenWithdrawal_disabledAtCutoff_succeeds(uint256 _cutoff) external {
        vm.prank(moduleOwner);
        module.disable(_cutoff);

        vm.expectCall(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), 0);
        vm.prank(address(optimismPortal2));
        assertTrue(
            module.screenWithdrawal(
                BridgeHookTypes.Item({
                    direction: BridgeHookTypes.Direction.Withdrawal,
                    asset: BridgeHookTypes.Asset.ETH,
                    from: alice,
                    aliased: false,
                    to: bob,
                    ethAmount: 1 ether,
                    l2Value: 0,
                    gasLimit: 100_000,
                    isCreation: false,
                    data: hex"",
                    nonce: _cutoff
                }),
                0
            )
        );
    }

    /// @notice Tests that screenWithdrawal skips the policy when disabled above the cutoff nonce.
    function testFuzz_screenWithdrawal_disabledAboveCutoff_succeeds(uint256 _cutoff, uint256 _nonce) external {
        _cutoff = bound(_cutoff, 0, type(uint256).max - 1);
        _nonce = bound(_nonce, _cutoff + 1, type(uint256).max);
        vm.prank(moduleOwner);
        module.disable(_cutoff);

        vm.expectCall(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), 0);
        vm.prank(address(optimismPortal2));
        assertTrue(
            module.screenWithdrawal(
                BridgeHookTypes.Item({
                    direction: BridgeHookTypes.Direction.Withdrawal,
                    asset: BridgeHookTypes.Asset.ETH,
                    from: alice,
                    aliased: false,
                    to: bob,
                    ethAmount: 1 ether,
                    l2Value: 0,
                    gasLimit: 100_000,
                    isCreation: false,
                    data: hex"",
                    nonce: _nonce
                }),
                0
            )
        );
    }

    /// @notice Tests that screenWithdrawal calls the policy below the cutoff when disabled.
    function testFuzz_screenWithdrawal_disabledBelowCutoff_succeeds(uint256 _cutoff, uint256 _nonce) external {
        _cutoff = bound(_cutoff, 1, type(uint256).max);
        _nonce = bound(_nonce, 0, _cutoff - 1);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: _nonce
        });
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });
        vm.prank(moduleOwner);
        module.disable(_cutoff);

        bytes memory policyCall = abi.encodeCall(IPolicy.screenWithdrawal, (item, parties, 0, 0));
        vm.mockCall(address(policy), policyCall, abi.encode(false));
        vm.expectCall(address(policy), policyCall, 1);
        vm.prank(address(optimismPortal2));
        assertFalse(module.screenWithdrawal(item, 0));
    }
}

/// @title ComplianceModule_HoldDeposit_Test
/// @notice Tests the `holdDeposit` function of the `ComplianceModule` contract.
contract ComplianceModule_HoldDeposit_Test is ComplianceModule_TestInit {
    using stdStorage for StdStorage;

    /// @notice Tests that holdDeposit reverts for a caller other than the Portal.
    function testFuzz_holdDeposit_notPortal_reverts(address _caller) external {
        vm.assume(_caller != address(optimismPortal2));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPortal.selector);
        vm.prank(_caller);
        module.holdDeposit(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Deposit,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 0,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that holdDeposit reverts when the module is not the Portal's bridge hook.
    function test_holdDeposit_notActiveHook_reverts() external {
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.bridgeHook, ()), abi.encode(makeAddr("otherHook"))
        );

        vm.expectRevert(IComplianceModule.ComplianceModule_NotActiveHook.selector);
        vm.prank(address(optimismPortal2));
        module.holdDeposit(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Deposit,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 0,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that holdDeposit records the hold time, emits Held and keeps the ETH.
    function testFuzz_holdDeposit_eth_succeeds(uint256 _ethAmount) external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: _ethAmount,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        vm.deal(address(optimismPortal2), _ethAmount);

        vm.expectCall(address(l1StandardBridge), abi.encodePacked(IL1StandardBridge.heldTokenKinds.selector), 0);
        vm.expectEmit(address(module));
        emit Held(id, item);
        vm.prank(address(optimismPortal2));
        module.holdDeposit{ value: _ethAmount }(item);

        assertEq(module.items(id).heldAt, block.timestamp);
        assertEq(address(module).balance, _ethAmount);
    }

    /// @notice Tests that holdDeposit reverts when msg.value differs from the item's ethAmount.
    function testFuzz_holdDeposit_ethAmountMismatch_reverts(uint256 _ethAmount, uint256 _value) external {
        vm.assume(_value != _ethAmount);
        vm.deal(address(optimismPortal2), _value);

        vm.expectRevert(IComplianceModule.ComplianceModule_ETHAmountMismatch.selector);
        vm.prank(address(optimismPortal2));
        module.holdDeposit{ value: _value }(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Deposit,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: _ethAmount,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that holdDeposit adds the escrowed amount to heldTokens for the token pair.
    function testFuzz_holdDeposit_escrowedToken_succeeds(uint256 _amount, uint256 _existing) external {
        _amount = bound(_amount, 0, type(uint256).max - _existing);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, _amount, hex"")
                    )
                )
            ),
            nonce: 1
        });
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(_existing);
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.heldTokenKinds, (BridgeHookHashing.hashItem(item))),
            abi.encode(BridgeHookTypes.TokenKind.Escrowed)
        );

        deal(address(L1Token), address(module), _existing + _amount);

        vm.prank(address(optimismPortal2));
        module.holdDeposit(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), _existing + _amount);
    }

    /// @notice Tests that holdDeposit reverts when the token balance does not cover heldTokens.
    function testFuzz_holdDeposit_insufficientBalance_reverts(
        uint256 _amount,
        uint256 _existing,
        uint256 _balance
    )
        external
    {
        _existing = bound(_existing, 0, type(uint256).max - 1);
        _amount = bound(_amount, 1, type(uint256).max - _existing);
        _balance = bound(_balance, 0, _existing + _amount - 1);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, _amount, hex"")
                    )
                )
            ),
            nonce: 1
        });
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(_existing);
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.heldTokenKinds, (BridgeHookHashing.hashItem(item))),
            abi.encode(BridgeHookTypes.TokenKind.Escrowed)
        );
        deal(address(L1Token), address(module), _balance);

        vm.expectRevert(IComplianceModule.ComplianceModule_InsufficientBalance.selector);
        vm.prank(address(optimismPortal2));
        module.holdDeposit(item);
    }

    /// @notice Tests that holdDeposit leaves heldTokens unchanged for a mintable token.
    function test_holdDeposit_mintableToken_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.heldTokenKinds, (BridgeHookHashing.hashItem(item))),
            abi.encode(BridgeHookTypes.TokenKind.Mintable)
        );

        vm.prank(address(optimismPortal2));
        module.holdDeposit(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 0);
    }
}

/// @title ComplianceModule_HoldWithdrawal_Test
/// @notice Tests the `holdWithdrawal` function of the `ComplianceModule` contract.
contract ComplianceModule_HoldWithdrawal_Test is ComplianceModule_TestInit {
    using stdStorage for StdStorage;

    /// @notice Tests that holdWithdrawal reverts for a caller other than the Portal.
    function testFuzz_holdWithdrawal_notPortal_reverts(address _caller) external {
        vm.assume(_caller != address(optimismPortal2));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPortal.selector);
        vm.prank(_caller);
        module.holdWithdrawal(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 0,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that holdWithdrawal reverts when the module is not the Portal's bridge hook.
    function test_holdWithdrawal_notActiveHook_reverts() external {
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.bridgeHook, ()), abi.encode(makeAddr("otherHook"))
        );

        vm.expectRevert(IComplianceModule.ComplianceModule_NotActiveHook.selector);
        vm.prank(address(optimismPortal2));
        module.holdWithdrawal(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 0,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that holdWithdrawal records the hold time, emits Held and keeps the ETH.
    function testFuzz_holdWithdrawal_eth_succeeds(uint256 _ethAmount) external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: _ethAmount,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        vm.deal(address(optimismPortal2), _ethAmount);

        vm.expectCall(address(l1StandardBridge), abi.encodePacked(IL1StandardBridge.heldTokenKinds.selector), 0);
        vm.expectEmit(address(module));
        emit Held(id, item);
        vm.prank(address(optimismPortal2));
        module.holdWithdrawal{ value: _ethAmount }(item);

        assertEq(module.items(id).heldAt, block.timestamp);
        assertEq(address(module).balance, _ethAmount);
    }

    /// @notice Tests that holdWithdrawal reverts when msg.value differs from the item's ethAmount.
    function testFuzz_holdWithdrawal_ethAmountMismatch_reverts(uint256 _ethAmount, uint256 _value) external {
        vm.assume(_value != _ethAmount);
        vm.deal(address(optimismPortal2), _value);

        vm.expectRevert(IComplianceModule.ComplianceModule_ETHAmountMismatch.selector);
        vm.prank(address(optimismPortal2));
        module.holdWithdrawal{ value: _value }(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: _ethAmount,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that holdWithdrawal leaves the recorded verdict's expiration time unchanged.
    function test_holdWithdrawal_recordedVerdict_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, 0, expiresAt);

        vm.prank(address(optimismPortal2));
        module.holdWithdrawal(item);

        assertEq(module.items(id).heldAt, block.timestamp);
        assertEq(module.items(id).expiresAt, expiresAt);
    }

    /// @notice Tests that holdWithdrawal adds the escrowed amount to heldTokens for the token pair.
    function testFuzz_holdWithdrawal_escrowedToken_succeeds(uint256 _amount, uint256 _existing) external {
        _amount = bound(_amount, 0, type(uint256).max - _existing);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: address(l1CrossDomainMessenger),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    Predeploys.L2_STANDARD_BRIDGE,
                    address(l1StandardBridge),
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L1Token), address(L2Token), alice, bob, _amount, hex"")
                    )
                )
            ),
            nonce: 1
        });
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(_existing);
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.heldTokenKinds, (BridgeHookHashing.hashItem(item))),
            abi.encode(BridgeHookTypes.TokenKind.Escrowed)
        );

        deal(address(L1Token), address(module), _existing + _amount);

        vm.prank(address(optimismPortal2));
        module.holdWithdrawal(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), _existing + _amount);
    }

    /// @notice Tests that holdWithdrawal reverts when the token balance does not cover heldTokens.
    function testFuzz_holdWithdrawal_insufficientBalance_reverts(
        uint256 _amount,
        uint256 _existing,
        uint256 _balance
    )
        external
    {
        _existing = bound(_existing, 0, type(uint256).max - 1);
        _amount = bound(_amount, 1, type(uint256).max - _existing);
        _balance = bound(_balance, 0, _existing + _amount - 1);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: address(l1CrossDomainMessenger),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    Predeploys.L2_STANDARD_BRIDGE,
                    address(l1StandardBridge),
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L1Token), address(L2Token), alice, bob, _amount, hex"")
                    )
                )
            ),
            nonce: 1
        });
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(_existing);
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.heldTokenKinds, (BridgeHookHashing.hashItem(item))),
            abi.encode(BridgeHookTypes.TokenKind.Escrowed)
        );
        deal(address(L1Token), address(module), _balance);

        vm.expectRevert(IComplianceModule.ComplianceModule_InsufficientBalance.selector);
        vm.prank(address(optimismPortal2));
        module.holdWithdrawal(item);
    }

    /// @notice Tests that holdWithdrawal leaves heldTokens unchanged for a mintable token.
    function test_holdWithdrawal_mintableToken_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: address(l1CrossDomainMessenger),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    Predeploys.L2_STANDARD_BRIDGE,
                    address(l1StandardBridge),
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L1Token), address(L2Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.heldTokenKinds, (BridgeHookHashing.hashItem(item))),
            abi.encode(BridgeHookTypes.TokenKind.Mintable)
        );

        vm.prank(address(optimismPortal2));
        module.holdWithdrawal(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 0);
    }
}

/// @title ComplianceModule_RecordVerdict_Test
/// @notice Tests the `recordVerdict` function of the `ComplianceModule` contract.
contract ComplianceModule_RecordVerdict_Test is ComplianceModule_TestInit {
    /// @notice Tests that recordVerdict reverts for a caller other than the policy.
    function testFuzz_recordVerdict_notPolicy_reverts(address _caller) external {
        vm.assume(_caller != address(policy));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPolicy.selector);
        vm.prank(_caller);
        module.recordVerdict(keccak256("item"), uint64(block.timestamp + 1 days));
    }

    /// @notice Tests that recordVerdict reverts for a zero expiration time.
    function test_recordVerdict_zeroExpiry_reverts() external {
        vm.expectRevert(IComplianceModule.ComplianceModule_ZeroExpiry.selector);
        vm.prank(address(policy));
        module.recordVerdict(keccak256("item"), 0);
    }

    /// @notice Tests that recordVerdict stores a nonzero expiration time for an item that is not held.
    function testFuzz_recordVerdict_itemNotHeld_succeeds(bytes32 _id, uint64 _expiresAt) external {
        _expiresAt = uint64(bound(_expiresAt, 1, type(uint64).max));
        vm.expectEmit(address(module));
        emit VerdictRecorded(_id, _expiresAt);
        vm.prank(address(policy));
        module.recordVerdict(_id, _expiresAt);

        assertEq(module.items(_id).heldAt, 0);
        assertEq(module.items(_id).expiresAt, _expiresAt);
    }

    /// @notice Tests that recordVerdict keeps the hold time of a held item.
    function test_recordVerdict_heldItem_succeeds() external {
        bytes32 id = keccak256("item");
        uint64 heldAt = uint64(block.timestamp);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, heldAt, 0);

        vm.prank(address(policy));
        module.recordVerdict(id, expiresAt);

        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, expiresAt);
    }

    /// @notice Tests that only the current policy can record a verdict after a policy change.
    function test_recordVerdict_replacedPolicy_succeeds() external {
        IPolicy newPolicy = IPolicy(makeAddr("newPolicy"));
        bytes32 id = keccak256("item");
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        vm.prank(moduleOwner);
        module.setPolicy(newPolicy);

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPolicy.selector);
        vm.prank(address(policy));
        module.recordVerdict(id, expiresAt);

        vm.prank(address(newPolicy));
        module.recordVerdict(id, expiresAt);
        assertEq(module.items(id).expiresAt, expiresAt);
    }
}

/// @title ComplianceModule_RevokeVerdict_Test
/// @notice Tests the `revokeVerdict` function of the `ComplianceModule` contract.
contract ComplianceModule_RevokeVerdict_Test is ComplianceModule_TestInit {
    /// @notice Tests that revokeVerdict reverts for a caller other than the policy.
    function testFuzz_revokeVerdict_notPolicy_reverts(address _caller) external {
        vm.assume(_caller != address(policy));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPolicy.selector);
        vm.prank(_caller);
        module.revokeVerdict(keccak256("item"));
    }

    /// @notice Tests that revokeVerdict clears the expiration time and keeps the hold time.
    function test_revokeVerdict_succeeds() external {
        bytes32 id = keccak256("item");
        uint64 heldAt = uint64(block.timestamp);
        _storeRecord(id, heldAt, uint64(block.timestamp + 1 days));

        vm.expectEmit(address(module));
        emit VerdictRevoked(id);
        vm.prank(address(policy));
        module.revokeVerdict(id);

        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, 0);
    }

    /// @notice Tests that revokeVerdict emits VerdictRevoked even when no record exists.
    function test_revokeVerdict_noRecord_succeeds() external {
        bytes32 id = keccak256("item");

        vm.expectEmit(address(module));
        emit VerdictRevoked(id);
        vm.prank(address(policy));
        module.revokeVerdict(id);
    }
}

/// @title ComplianceModule_SetPolicy_Test
/// @notice Tests the `setPolicy` function of the `ComplianceModule` contract.
contract ComplianceModule_SetPolicy_Test is ComplianceModule_TestInit {
    /// @notice Tests that setPolicy reverts for a caller other than the owner.
    function testFuzz_setPolicy_notOwner_reverts(address _caller) external {
        vm.assume(_caller != moduleOwner);

        vm.expectRevert(bytes.concat(IComplianceModule.OwnableUnauthorizedAccount.selector, abi.encode(_caller)));
        vm.prank(_caller);
        module.setPolicy(IPolicy(makeAddr("newPolicy")));
    }

    /// @notice Tests that setPolicy sets the policy, emits PolicySet and keeps item records.
    function testFuzz_setPolicy_succeeds(IPolicy _newPolicy) external {
        vm.assume(address(_newPolicy) != address(0));
        bytes32 id = keccak256("item");
        uint64 heldAt = uint64(block.timestamp);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, heldAt, expiresAt);

        vm.expectEmit(address(module));
        emit PolicySet(policy, _newPolicy);
        vm.prank(moduleOwner);
        module.setPolicy(_newPolicy);

        assertEq(address(module.policy()), address(_newPolicy));
        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, expiresAt);
    }

    /// @notice Tests that setPolicy reverts when the policy address is zero.
    function test_setPolicy_zeroAddress_reverts() external {
        vm.expectRevert(IComplianceModule.ComplianceModule_ZeroAddress.selector);
        vm.prank(moduleOwner);
        module.setPolicy(IPolicy(address(0)));
    }
}

/// @title ComplianceModule_Disable_Test
/// @notice Tests the `disable` function of the `ComplianceModule` contract.
contract ComplianceModule_Disable_Test is ComplianceModule_TestInit {
    /// @notice Tests that disable reverts for a caller other than the owner.
    function testFuzz_disable_notOwner_reverts(address _caller) external {
        vm.assume(_caller != moduleOwner);

        vm.expectRevert(bytes.concat(IComplianceModule.OwnableUnauthorizedAccount.selector, abi.encode(_caller)));
        vm.prank(_caller);
        module.disable(0);
    }

    /// @notice Tests that disable sets the cutoff and emits Disabled on each call.
    function testFuzz_disable_succeeds(uint256 _cutoff, uint256 _newCutoff) external {
        vm.expectEmit(address(module));
        emit Disabled(_cutoff);
        vm.prank(moduleOwner);
        module.disable(_cutoff);

        assertTrue(module.disabled());
        assertEq(module.cutoffL2Nonce(), _cutoff);

        vm.expectEmit(address(module));
        emit Disabled(_newCutoff);
        vm.prank(moduleOwner);
        module.disable(_newCutoff);

        assertTrue(module.disabled());
        assertEq(module.cutoffL2Nonce(), _newCutoff);
    }
}

/// @title ComplianceModule_Enable_Test
/// @notice Tests the `enable` function of the `ComplianceModule` contract.
contract ComplianceModule_Enable_Test is ComplianceModule_TestInit {
    /// @notice Tests that enable reverts for a caller other than the owner.
    function testFuzz_enable_notOwner_reverts(address _caller) external {
        vm.assume(_caller != moduleOwner);

        vm.expectRevert(bytes.concat(IComplianceModule.OwnableUnauthorizedAccount.selector, abi.encode(_caller)));
        vm.prank(_caller);
        module.enable();
    }

    /// @notice Tests that enable clears the cutoff and emits Enabled. A second call succeeds.
    function test_enable_succeeds() external {
        vm.prank(moduleOwner);
        module.disable(100);

        vm.expectEmit(address(module));
        emit Enabled();
        vm.prank(moduleOwner);
        module.enable();

        assertFalse(module.disabled());
        assertEq(module.cutoffL2Nonce(), 0);

        vm.expectEmit(address(module));
        emit Enabled();
        vm.prank(moduleOwner);
        module.enable();
    }
}

/// @title ComplianceModule_CompleteDeposit_Test
/// @notice Tests the `completeDeposit` function of the `ComplianceModule` contract.
contract ComplianceModule_CompleteDeposit_Test is ComplianceModule_TestInit {
    using stdStorage for StdStorage;

    /// @notice Calldata for the reentrant module call from screenRelease. Empty disables the call.
    bytes internal reentry;

    /// @notice Return data of the module call that screenRelease makes.
    bytes internal reentryReturnData;

    /// @notice Approves the release as the policy. Calls the module first when reentry is set.
    function screenRelease(BridgeHookTypes.Item calldata, IPolicy.Parties calldata, uint64) external returns (bool) {
        if (reentry.length > 0) {
            bytes memory data = reentry;
            delete reentry;
            (, reentryReturnData) = address(module).call(data);
        }
        return true;
    }

    /// @notice Tests that any caller can complete a held and cleared ETH deposit.
    function testFuzz_completeDeposit_eth_succeeds(address _caller, uint256 _ethAmount, uint64 _expiresAt) external {
        _ethAmount = bound(_ethAmount, 0, type(uint128).max);
        _expiresAt = uint64(bound(_expiresAt, 1, type(uint64).max));
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: _ethAmount,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });
        _storeRecord(id, uint64(block.timestamp), _expiresAt);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        vm.deal(address(module), _ethAmount);

        bytes memory policyCall = abi.encodeCall(IPolicy.screenRelease, (item, parties, _expiresAt));
        vm.mockCall(address(policy), policyCall, abi.encode(true));
        vm.expectCall(address(policy), policyCall, 1);
        vm.expectCall(
            address(optimismPortal2), _ethAmount, abi.encodeCall(IOptimismPortal2.completeDepositTransaction, (item)), 1
        );
        vm.expectCall(address(l1StandardBridge), abi.encodePacked(IL1StandardBridge.heldTokenKinds.selector), 0);
        vm.expectEmit(address(module));
        emit Completed(id);
        vm.prank(_caller);
        module.completeDeposit(item);

        assertEq(module.items(id).heldAt, 0);
        assertEq(module.items(id).expiresAt, 0);
        assertFalse(optimismPortal2.heldDeposits(id));
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that completeDeposit restores escrow, reduces heldTokens and clears allowance.
    function testFuzz_completeDeposit_escrowedToken_succeeds(uint256 _amount, uint256 _existing) external {
        // Exclude infinite allowance because this test checks that the transfer consumes it.
        _amount = bound(_amount, 0, type(uint256).max - 1);
        _existing = bound(_existing, 0, type(uint256).max - _amount);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, _amount, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Escrowed));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(_existing + _amount);
        deal(address(L1Token), address(module), _amount);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));

        module.completeDeposit(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), _existing);
        assertEq(L1Token.balanceOf(address(module)), 0);
        assertEq(L1Token.allowance(address(module), address(l1StandardBridge)), 0);
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), _amount);
    }

    /// @notice Tests that completeDeposit skips approval and heldTokens for a Mintable token.
    function test_completeDeposit_mintableToken_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Mintable));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(100);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));

        vm.expectCall(address(L1Token), abi.encodePacked(L1Token.approve.selector), 0);
        module.completeDeposit(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 100);
    }

    /// @notice Tests that completeDeposit reverts for an item that is not held.
    function test_completeDeposit_notHeld_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        _storeRecord(BridgeHookHashing.hashItem(item), 0, uint64(block.timestamp + 1 days));

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(item);
    }

    /// @notice Tests that completeDeposit reverts for a held item without a verdict.
    function test_completeDeposit_notCleared_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        _storeRecord(BridgeHookHashing.hashItem(item), uint64(block.timestamp), 0);

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotCleared.selector);
        module.completeDeposit(item);
    }

    /// @notice Tests that completion of a held item with one field changed reverts.
    function testFuzz_completeDeposit_changedField_reverts(uint256 _field, uint256 _value) external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"aa",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        uint256 f = bound(_field, 0, 10);
        BridgeHookTypes.Item memory changed = BridgeHookTypes.Item({
            direction: f == 0 ? BridgeHookTypes.Direction.Withdrawal : item.direction,
            asset: f == 1 ? BridgeHookTypes.Asset.ERC20 : item.asset,
            from: f == 2 ? address(uint160(_value)) : item.from,
            aliased: f == 3 ? !item.aliased : item.aliased,
            to: f == 4 ? address(uint160(_value)) : item.to,
            ethAmount: f == 5 ? _value : item.ethAmount,
            l2Value: f == 6 ? _value : item.l2Value,
            gasLimit: f == 7 ? _value : item.gasLimit,
            isCreation: f == 8 ? !item.isCreation : item.isCreation,
            data: f == 9 ? abi.encode(_value) : item.data,
            nonce: f == 10 ? _value : item.nonce
        });
        // Use the spec's identifier to keep this check independent of the hashing library.
        vm.assume(keccak256(abi.encode(changed)) != keccak256(abi.encode(item)));

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(changed);

        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        module.completeDeposit(item);
        assertEq(module.items(id).heldAt, 0);
    }

    /// @notice Tests that a policy refusal reverts completeDeposit and keeps the item held.
    function test_completeDeposit_policyRefuses_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        uint64 heldAt = uint64(block.timestamp);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, heldAt, expiresAt);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Escrowed));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(100);
        deal(address(L1Token), address(module), 100);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(false));

        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeDeposit(item);

        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, expiresAt);
        assertEq(L1Token.balanceOf(address(module)), 100);
        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 100);
        assertTrue(optimismPortal2.heldDeposits(id));
    }

    /// @notice Tests that completeDeposit keeps the policy's revert data and the item held.
    function test_completeDeposit_policyReverts_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        uint64 heldAt = uint64(block.timestamp);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, heldAt, expiresAt);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Escrowed));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(100);
        deal(address(L1Token), address(module), 100);
        vm.mockCallRevert(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), hex"c0ffee");

        vm.expectRevert(bytes(hex"c0ffee"));
        module.completeDeposit(item);

        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, expiresAt);
        assertEq(L1Token.balanceOf(address(module)), 100);
        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 100);
        assertTrue(optimismPortal2.heldDeposits(id));
    }

    /// @notice Tests that completeDeposit keeps restoreERC20's revert data and the item held.
    function test_completeDeposit_restoreFails_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L2Token), address(L1Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        uint64 heldAt = uint64(block.timestamp);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, heldAt, expiresAt);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Escrowed));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(100);
        deal(address(L1Token), address(module), 100);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.mockCallRevert(
            address(l1StandardBridge), abi.encodePacked(IL1StandardBridge.restoreERC20.selector), hex"c0ffee"
        );

        vm.expectRevert(bytes(hex"c0ffee"));
        module.completeDeposit(item);

        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, expiresAt);
        assertEq(L1Token.balanceOf(address(module)), 100);
        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 100);
        assertTrue(optimismPortal2.heldDeposits(id));
    }

    /// @notice Tests that a second completeDeposit for the same item reverts.
    function test_completeDeposit_twice_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        module.completeDeposit(item);

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(item);
    }

    /// @notice Tests that completeDeposit succeeds while the Portal is paused.
    function test_completeDeposit_paused_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));
        assertTrue(optimismPortal2.paused());

        vm.expectEmit(address(module));
        emit Completed(id);
        module.completeDeposit(item);
    }

    /// @notice Tests that a policy cannot complete the same deposit again while screenRelease runs.
    function test_completeDeposit_reentryFromPolicy_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        vm.prank(moduleOwner);
        module.setPolicy(IPolicy(address(this)));
        reentry = abi.encodeCall(IComplianceModule.completeDeposit, (item));

        module.completeDeposit(item);

        assertEq(reentryReturnData, bytes.concat(IComplianceModule.ComplianceModule_ItemNotHeld.selector));
    }

    /// @notice Tests that the token's transferFrom cannot complete the same deposit again.
    function test_completeDeposit_reentryFromToken_reverts() external {
        ComplianceModule_ReentrantERC20_Harness token = new ComplianceModule_ReentrantERC20_Harness();
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    0,
                    address(l1StandardBridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20, (address(L2Token), address(token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldDeposits.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Escrowed));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(token))
            .with_key(address(L2Token))
            .checked_write(100);
        deal(address(token), address(module), 100);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        token.setReentry(address(module), abi.encodeCall(IComplianceModule.completeDeposit, (item)));

        module.completeDeposit(item);

        assertEq(token.reentryReturnData(), bytes.concat(IComplianceModule.ComplianceModule_ItemNotHeld.selector));
    }
}

/// @title ComplianceModule_CompleteWithdrawal_Test
/// @notice Tests the `completeWithdrawal` function of the `ComplianceModule` contract.
contract ComplianceModule_CompleteWithdrawal_Test is ComplianceModule_TestInit {
    using stdStorage for StdStorage;

    /// @notice Calldata for the reentrant module call from onWithdrawal. Empty disables the call.
    bytes internal reentry;

    /// @notice Return data of the module call that onWithdrawal makes.
    bytes internal reentryReturnData;

    /// @notice Withdrawal target. Calls the module when reentry is set.
    function onWithdrawal() external payable {
        if (reentry.length > 0) {
            bytes memory data = reentry;
            delete reentry;
            (, reentryReturnData) = address(module).call(data);
        }
    }

    /// @notice Tests that any caller can complete a held and cleared ETH withdrawal.
    function testFuzz_completeWithdrawal_eth_succeeds(address _caller, uint256 _ethAmount, uint64 _expiresAt) external {
        _ethAmount = bound(_ethAmount, 0, type(uint128).max);
        _expiresAt = uint64(bound(_expiresAt, 1, type(uint64).max));
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: _ethAmount,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        IPolicy.Parties memory parties = IPolicy.Parties({ sender: alice, recipient: bob });
        _storeRecord(id, uint64(block.timestamp), _expiresAt);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldWithdrawals.selector)
            .with_key(id)
            .checked_write(true);
        vm.deal(address(module), _ethAmount);
        uint256 bobBalance = bob.balance;

        bytes memory policyCall = abi.encodeCall(IPolicy.screenRelease, (item, parties, _expiresAt));
        vm.mockCall(address(policy), policyCall, abi.encode(true));
        vm.expectCall(address(policy), policyCall, 1);
        vm.expectCall(
            address(optimismPortal2),
            _ethAmount,
            abi.encodeCall(IOptimismPortal2.completeWithdrawalTransaction, (item)),
            1
        );
        vm.expectCall(address(l1StandardBridge), abi.encodePacked(IL1StandardBridge.heldTokenKinds.selector), 0);
        vm.expectEmit(address(module));
        emit Completed(id);
        vm.prank(_caller);
        module.completeWithdrawal(item);

        assertEq(module.items(id).heldAt, 0);
        assertEq(module.items(id).expiresAt, 0);
        assertFalse(optimismPortal2.heldWithdrawals(id));
        assertEq(address(module).balance, 0);
        assertEq(bob.balance, bobBalance + _ethAmount);
    }

    /// @notice Tests that completeWithdrawal pays out, reduces heldTokens and clears allowance.
    function testFuzz_completeWithdrawal_escrowedToken_succeeds(uint256 _amount, uint256 _existing) external {
        // Exclude infinite allowance because this test checks that the transfer consumes it.
        _amount = bound(_amount, 0, type(uint256).max - 1);
        _existing = bound(_existing, 0, type(uint256).max - _amount);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: address(l1CrossDomainMessenger),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 500_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    Encoding.encodeVersionedNonce(0, 1),
                    Predeploys.L2_STANDARD_BRIDGE,
                    address(l1StandardBridge),
                    0,
                    200_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(L1Token), address(L2Token), alice, bob, _amount, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldWithdrawals.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Escrowed));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .checked_write(_existing + _amount);
        deal(address(L1Token), address(module), _amount);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));

        module.completeWithdrawal(item);

        assertEq(module.heldTokens(address(L1Token), address(L2Token)), _existing);
        assertEq(L1Token.balanceOf(address(module)), 0);
        assertEq(L1Token.allowance(address(module), address(l1StandardBridge)), 0);
        assertEq(L1Token.balanceOf(bob), _amount);
    }

    /// @notice Tests that completeWithdrawal skips approval and heldTokens for a Mintable token.
    function test_completeWithdrawal_mintableToken_succeeds() external {
        ERC20 mintableToken =
            ERC20(l1OptimismMintableERC20Factory.createStandardL2Token(address(NativeL2Token), "L1 Mintable", "L1M"));
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: address(l1CrossDomainMessenger),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 500_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    Encoding.encodeVersionedNonce(0, 1),
                    Predeploys.L2_STANDARD_BRIDGE,
                    address(l1StandardBridge),
                    0,
                    200_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (address(mintableToken), address(NativeL2Token), alice, bob, 100, hex"")
                    )
                )
            ),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldWithdrawals.selector)
            .with_key(id)
            .checked_write(true);
        stdstore.target(address(l1StandardBridge))
            .sig(IL1StandardBridge.heldTokenKinds.selector)
            .with_key(id)
            .checked_write(uint256(BridgeHookTypes.TokenKind.Mintable));
        stdstore.target(address(module))
            .sig(IComplianceModule.heldTokens.selector)
            .with_key(address(mintableToken))
            .with_key(address(NativeL2Token))
            .checked_write(100);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));

        vm.expectCall(address(mintableToken), abi.encodePacked(mintableToken.approve.selector), 0);
        module.completeWithdrawal(item);

        assertEq(module.heldTokens(address(mintableToken), address(NativeL2Token)), 100);
        assertEq(mintableToken.balanceOf(bob), 100);
    }

    /// @notice Tests that completeWithdrawal reverts when the target reverts. The item stays held.
    function test_completeWithdrawal_targetFails_reverts() external {
        address target = makeAddr("target");
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: target,
            ethAmount: 1 ether,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        uint64 heldAt = uint64(block.timestamp);
        uint64 expiresAt = uint64(block.timestamp + 1 days);
        _storeRecord(id, heldAt, expiresAt);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldWithdrawals.selector)
            .with_key(id)
            .checked_write(true);
        vm.deal(address(module), 1 ether);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.mockCallRevert(target, bytes(hex""), bytes(hex""));

        vm.expectRevert(IOptimismPortal2.OptimismPortal_TargetCallFailed.selector);
        module.completeWithdrawal(item);

        assertEq(module.items(id).heldAt, heldAt);
        assertEq(module.items(id).expiresAt, expiresAt);
        assertEq(address(module).balance, 1 ether);
        assertTrue(optimismPortal2.heldWithdrawals(id));
    }

    /// @notice Tests that disable and enable do not bypass verdict or policy approval at release.
    function test_completeWithdrawal_disabledAndEnabled_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), 0);
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldWithdrawals.selector)
            .with_key(id)
            .checked_write(true);
        vm.prank(moduleOwner);
        module.disable(1);

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotCleared.selector);
        module.completeWithdrawal(item);

        vm.prank(address(policy));
        module.recordVerdict(id, uint64(block.timestamp + 1 days));
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(false));

        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeWithdrawal(item);

        vm.prank(moduleOwner);
        module.enable();

        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeWithdrawal(item);
    }

    /// @notice Tests that completeWithdrawal reverts with the pause error for an item not held.
    function test_completeWithdrawal_paused_reverts() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        vm.expectRevert(IComplianceModule.ComplianceModule_Paused.selector);
        module.completeWithdrawal(
            BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction.Withdrawal,
                asset: BridgeHookTypes.Asset.ETH,
                from: alice,
                aliased: false,
                to: bob,
                ethAmount: 0,
                l2Value: 0,
                gasLimit: 100_000,
                isCreation: false,
                data: hex"",
                nonce: 1
            })
        );
    }

    /// @notice Tests that a target called by the Portal cannot complete the same withdrawal again.
    function test_completeWithdrawal_reentryFromTarget_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: address(this),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 500_000,
            isCreation: false,
            data: abi.encodeCall(this.onWithdrawal, ()),
            nonce: 1
        });
        bytes32 id = BridgeHookHashing.hashItem(item);
        _storeRecord(id, uint64(block.timestamp), uint64(block.timestamp + 1 days));
        stdstore.target(address(optimismPortal2))
            .sig(IOptimismPortal2.heldWithdrawals.selector)
            .with_key(id)
            .checked_write(true);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        reentry = abi.encodeCall(IComplianceModule.completeWithdrawal, (item));

        module.completeWithdrawal(item);

        assertEq(reentryReturnData, bytes.concat(IComplianceModule.ComplianceModule_ItemNotHeld.selector));
    }
}

/// @title ComplianceModule_Integration_Test
/// @notice Tests the module as the hook of the real Portal and bridge, with proven withdrawals.
contract ComplianceModule_Integration_Test is OptimismPortal2_TestInit, ComplianceModule_TestInit {
    function setUp() public override(OptimismPortal2_TestInit, ComplianceModule_TestInit) {
        super.setUp();
    }

    /// @notice Tests that a completed held ETH deposit matches one approved during screening.
    function test_completeDeposit_ethDeposit_succeeds() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 1 ether,
            l2Value: 1 ether,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 0
        });
        vm.deal(alice, 1 ether);
        uint256 snapshot = vm.snapshotState();

        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited(alice, bob, 1 ether, 1 ether, 100_000, false, hex"");
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: 1 ether }(bob, 1 ether, 100_000, false, hex"");
        uint256 portalBalance = address(optimismPortal2).balance;
        uint256 lockboxBalance = address(ethLockbox).balance;

        vm.revertToState(snapshot);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), abi.encode(false));
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: 1 ether }(bob, 1 ether, 100_000, false, hex"");
        vm.prank(address(policy));
        module.recordVerdict(BridgeHookHashing.hashItem(item), uint64(block.timestamp + 1 days));
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited(alice, bob, 1 ether, 1 ether, 100_000, false, hex"");
        vm.prank(makeAddr("completer"));
        module.completeDeposit(item);

        assertEq(address(optimismPortal2).balance, portalBalance);
        assertEq(address(ethLockbox).balance, lockboxBalance);
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that a completed held token deposit matches one approved during screening.
    function test_completeDeposit_escrowedTokenDeposit_succeeds() external {
        bytes memory message = abi.encodeCall(
            IStandardBridge.finalizeBridgeERC20, (address(L2Token), address(L1Token), alice, bob, 100, hex"")
        );
        bytes memory data = Encoding.encodeCrossDomainMessageV1(
            l1CrossDomainMessenger.messageNonce(),
            address(l1StandardBridge),
            Predeploys.L2_STANDARD_BRIDGE,
            0,
            200_000,
            message
        );
        uint64 gasLimit = l1CrossDomainMessenger.baseGas(message, 200_000);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: gasLimit,
            isCreation: false,
            data: data,
            nonce: 0
        });
        deal(address(L1Token), alice, 100);
        vm.prank(alice);
        L1Token.approve(address(l1StandardBridge), 100);
        uint256 snapshot = vm.snapshotState();

        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited(item.from, Predeploys.L2_CROSS_DOMAIN_MESSENGER, 0, 0, gasLimit, false, data);
        vm.prank(alice, alice);
        l1StandardBridge.bridgeERC20To(address(L1Token), address(L2Token), bob, 100, 200_000, hex"");

        vm.revertToState(snapshot);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), abi.encode(false));
        vm.prank(alice, alice);
        l1StandardBridge.bridgeERC20To(address(L1Token), address(L2Token), bob, 100, 200_000, hex"");
        vm.prank(address(policy));
        module.recordVerdict(BridgeHookHashing.hashItem(item), uint64(block.timestamp + 1 days));
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited(item.from, Predeploys.L2_CROSS_DOMAIN_MESSENGER, 0, 0, gasLimit, false, data);
        vm.prank(makeAddr("completer"));
        module.completeDeposit(item);

        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), 100);
        assertEq(L1Token.balanceOf(address(l1StandardBridge)), 100);
        assertEq(L1Token.balanceOf(address(module)), 0);
        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 0);
    }

    /// @notice Tests that a completed held ETH withdrawal matches one approved during screening.
    function test_completeWithdrawal_ethWithdrawal_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = Types.WithdrawalTransaction({
            nonce: 0, sender: alice, target: bob, value: 100, gasLimit: 100_000, data: hex"aa"
        });
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 100,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"aa",
            nonce: 0
        });
        bytes32 withdrawalHash = _prepareWithdrawalForFinalization(withdrawal);
        uint256 snapshot = vm.snapshotState();

        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), abi.encode(true));
        vm.expectCall(bob, 100, hex"aa");
        vm.expectEmit(address(optimismPortal2));
        emit WithdrawalFinalized(withdrawalHash, true);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        uint256 bobBalance = bob.balance;
        uint256 portalBalance = address(optimismPortal2).balance;
        uint256 lockboxBalance = address(ethLockbox).balance;

        vm.revertToState(snapshot);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), abi.encode(false));
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        vm.prank(address(policy));
        module.recordVerdict(BridgeHookHashing.hashItem(item), uint64(block.timestamp + 1 days));
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emit WithdrawalFinalized(withdrawalHash, true);
        vm.prank(makeAddr("completer"));
        module.completeWithdrawal(item);

        assertEq(bob.balance, bobBalance);
        assertEq(address(optimismPortal2).balance, portalBalance);
        assertEq(address(ethLockbox).balance, lockboxBalance);
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that a completed held token withdrawal matches one approved during screening.
    function test_completeWithdrawal_escrowedTokenWithdrawal_succeeds() external {
        bytes memory message = abi.encodeCall(
            IStandardBridge.finalizeBridgeERC20, (address(L1Token), address(L2Token), alice, bob, 100, hex"")
        );
        bytes memory data = Encoding.encodeCrossDomainMessageV1(
            Encoding.encodeVersionedNonce(0, 1),
            Predeploys.L2_STANDARD_BRIDGE,
            address(l1StandardBridge),
            0,
            200_000,
            message
        );
        uint64 gasLimit = l1CrossDomainMessenger.baseGas(message, 200_000);
        Types.WithdrawalTransaction memory withdrawal = Types.WithdrawalTransaction({
            nonce: 0,
            sender: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            target: address(l1CrossDomainMessenger),
            value: 0,
            gasLimit: gasLimit,
            data: data
        });
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: address(l1CrossDomainMessenger),
            ethAmount: 0,
            l2Value: 0,
            gasLimit: gasLimit,
            isCreation: false,
            data: data,
            nonce: 0
        });
        deal(address(L1Token), alice, 100);
        vm.prank(alice);
        L1Token.approve(address(l1StandardBridge), 100);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenDeposit.selector), abi.encode(true));
        vm.prank(alice, alice);
        l1StandardBridge.bridgeERC20To(address(L1Token), address(L2Token), alice, 100, 200_000, hex"");
        bytes32 withdrawalHash = _prepareWithdrawalForFinalization(withdrawal);
        uint256 snapshot = vm.snapshotState();

        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emit WithdrawalFinalized(withdrawalHash, true);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        assertEq(L1Token.balanceOf(bob), 100);

        vm.revertToState(snapshot);
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenWithdrawal.selector), abi.encode(false));
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        vm.prank(address(policy));
        module.recordVerdict(BridgeHookHashing.hashItem(item), uint64(block.timestamp + 1 days));
        vm.mockCall(address(policy), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(true));
        vm.expectEmit(address(optimismPortal2));
        emit WithdrawalFinalized(withdrawalHash, true);
        vm.prank(makeAddr("completer"));
        module.completeWithdrawal(item);

        assertEq(L1Token.balanceOf(bob), 100);
        assertEq(L1Token.balanceOf(address(l1StandardBridge)), 0);
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), 0);
        assertEq(L1Token.balanceOf(address(module)), 0);
        assertEq(module.heldTokens(address(L1Token), address(L2Token)), 0);
    }

    /// @notice Tests that a removed hook's holdWithdrawal reverts inside a failed withdrawal.
    function test_holdWithdrawal_formerHook_reverts() external {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: bob,
            ethAmount: 100,
            l2Value: 0,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: 0
        });
        Types.WithdrawalTransaction memory withdrawal = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: address(module),
            value: 100,
            gasLimit: 200_000,
            data: abi.encodeCall(IBridgeHook.holdWithdrawal, (item))
        });
        vm.prank(optimismPortal2.proxyAdminOwner());
        optimismPortal2.removeHook();
        bytes32 withdrawalHash = _prepareWithdrawalForFinalization(withdrawal);

        vm.expectCall(address(module), 100, abi.encodeCall(IBridgeHook.holdWithdrawal, (item)), 1);
        vm.expectEmit(address(optimismPortal2));
        emit WithdrawalFinalized(withdrawalHash, false);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);

        assertEq(module.items(BridgeHookHashing.hashItem(item)).heldAt, 0);
        assertEq(address(module).balance, 0);
    }
}

/// @title ComplianceModule_Uncategorized_Test
/// @notice Tests behavior of the `ComplianceModule` contract that spans several functions.
contract ComplianceModule_Uncategorized_Test is ComplianceModule_TestInit {
    /// @notice Tests that holdWithdrawal records and emits the expected ID of each fixture item.
    function test_hold_fixtureIds_succeeds() external {
        string memory json = vm.readFile("test/fixtures/bridge-hook-items.json");
        for (uint256 i; vm.keyExistsJson(json, string.concat(".vectors[", vm.toString(i), "]")); i++) {
            string memory key = string.concat(".vectors[", vm.toString(i), "].item.");
            BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
                direction: BridgeHookTypes.Direction(vm.parseJsonUint(json, string.concat(key, "direction"))),
                asset: BridgeHookTypes.Asset(vm.parseJsonUint(json, string.concat(key, "asset"))),
                from: vm.parseJsonAddress(json, string.concat(key, "from")),
                aliased: vm.parseJsonBool(json, string.concat(key, "aliased")),
                to: vm.parseJsonAddress(json, string.concat(key, "to")),
                ethAmount: vm.parseJsonUint(json, string.concat(key, "ethAmount")),
                l2Value: vm.parseJsonUint(json, string.concat(key, "l2Value")),
                gasLimit: vm.parseJsonUint(json, string.concat(key, "gasLimit")),
                isCreation: vm.parseJsonBool(json, string.concat(key, "isCreation")),
                data: vm.parseJsonBytes(json, string.concat(key, "data")),
                nonce: vm.parseJsonUint(json, string.concat(key, "nonce"))
            });
            bytes32 id = vm.parseJsonBytes32(json, string.concat(".vectors[", vm.toString(i), "].id"));
            vm.deal(address(optimismPortal2), item.ethAmount);

            vm.expectEmit(address(module));
            emit Held(id, item);
            vm.prank(address(optimismPortal2));
            module.holdWithdrawal{ value: item.ethAmount }(item);

            assertEq(module.items(id).heldAt, block.timestamp);
        }
    }
}

/// @title ComplianceModule_ReentrantERC20_Harness
/// @notice ERC20 test token that makes a configured callback during transferFrom, before
///         transferring tokens, and records the return data.
contract ComplianceModule_ReentrantERC20_Harness is ERC20 {
    /// @notice Address that the next transferFrom calls.
    address internal reentryTarget;

    /// @notice Calldata that the next transferFrom sends. Empty means no call.
    bytes internal reentry;

    /// @notice Return data of the call.
    bytes public reentryReturnData;

    constructor() ERC20("Reentrant Token", "RT") { }

    /// @notice Sets the call that the next transferFrom makes.
    function setReentry(address _target, bytes calldata _data) external {
        reentryTarget = _target;
        reentry = _data;
    }

    /// @notice Executes any configured callback once before transferring tokens.
    function transferFrom(address _from, address _to, uint256 _amount) public override returns (bool) {
        if (reentry.length > 0) {
            bytes memory data = reentry;
            delete reentry;
            (, reentryReturnData) = reentryTarget.call(data);
        }
        return super.transferFrom(_from, _to, _amount);
    }
}
