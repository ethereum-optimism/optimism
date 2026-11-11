// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { StdCheats } from "forge-std/StdCheats.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { ComplianceModule_TestInit } from "test/compliance/ComplianceModule.t.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";

/// @title ComplianceModule_User
/// @notice Holds, clears and completes items through the module, and changes its policy and
///         screening. Seeds the Portal and bridge state that precedes each hold.
contract ComplianceModule_User is StdUtils, StdCheats {
    Vm internal vm;
    IComplianceModule internal module;
    IOptimismPortal2 internal portal;
    IL1StandardBridge internal bridge;
    address internal l1CrossDomainMessenger;
    address internal l1Token;
    address[2] internal l2Tokens;
    address internal owner;
    IPolicy[3] internal policies;
    uint256 internal heldDepositsSlot;
    uint256 internal heldWithdrawalsSlot;
    uint256 internal heldTokenKindsSlot;

    /// @notice Source of a unique nonce for each item.
    uint256 internal nonce;

    /// @notice Items held so far, in hold order.
    BridgeHookTypes.Item[] internal heldItems;

    /// @notice Item identifiers in hold order, including completed items.
    bytes32[] public ids;

    /// @notice Indexes of held items that received a verdict, kept after revocation or completion.
    uint256[] internal verdictIndexes;

    /// @notice ETH that each held item carries.
    mapping(bytes32 => uint256) public ethAmountOf;

    /// @notice Token amount that each held escrowed token item carries. Zero for other items.
    mapping(bytes32 => uint256) public escrowedAmountOf;

    /// @notice Number of Completed events the module emitted for each item.
    mapping(bytes32 => uint256) public completedCount;

    /// @notice Whether a completion succeeded when it should have failed, or failed when it should
    ///         have succeeded.
    bool public completionMismatch;

    constructor(
        Vm _vm,
        IComplianceModule _module,
        IOptimismPortal2 _portal,
        IL1StandardBridge _bridge,
        address _l1CrossDomainMessenger,
        address _l1Token,
        address[2] memory _l2Tokens,
        address _owner,
        IPolicy[3] memory _policies
    ) {
        vm = _vm;
        module = _module;
        portal = _portal;
        bridge = _bridge;
        l1CrossDomainMessenger = _l1CrossDomainMessenger;
        l1Token = _l1Token;
        l2Tokens = _l2Tokens;
        owner = _owner;
        policies = _policies;
        heldDepositsSlot = ForgeArtifacts.getSlot("OptimismPortal2", "heldDeposits").slot;
        heldWithdrawalsSlot = ForgeArtifacts.getSlot("OptimismPortal2", "heldWithdrawals").slot;
        heldTokenKindsSlot = ForgeArtifacts.getSlot("L1StandardBridge", "heldTokenKinds").slot;
    }

    /// @notice Returns the total number of items held so far, including completed items.
    function idCount() external view returns (uint256) {
        return ids.length;
    }

    /// @notice Holds a new ETH deposit or withdrawal.
    function holdEth(bool _withdrawal, uint256 _amount) external {
        _amount = bound(_amount, 0, 1e30);
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: _withdrawal ? BridgeHookTypes.Direction.Withdrawal : BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: address(0xA11CE),
            aliased: false,
            to: address(0xB0B),
            ethAmount: _amount,
            l2Value: _withdrawal ? 0 : _amount,
            gasLimit: 100_000,
            isCreation: false,
            data: hex"",
            nonce: nonce++
        });
        _hold(item, BridgeHookTypes.TokenKind.None, 0);
    }

    /// @notice Holds a new escrowed or mintable token deposit for either L2 token.
    function holdTokenDeposit(bool _mintable, bool _secondPair, uint256 _amount) external {
        _amount = bound(_amount, 0, 1e30);
        uint256 itemNonce = nonce++;
        address l2Token = l2Tokens[_secondPair ? 1 : 0];
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ERC20,
            from: AddressAliasHelper.applyL1ToL2Alias(l1CrossDomainMessenger),
            aliased: true,
            to: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 200_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    Encoding.encodeVersionedNonce(uint240(itemNonce), 1),
                    address(bridge),
                    Predeploys.L2_STANDARD_BRIDGE,
                    0,
                    100_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (l2Token, l1Token, address(0xA11CE), address(0xB0B), _amount, hex"")
                    )
                )
            ),
            nonce: itemNonce
        });
        _hold(item, _mintable ? BridgeHookTypes.TokenKind.Mintable : BridgeHookTypes.TokenKind.Escrowed, _amount);
    }

    /// @notice Holds a new escrowed token withdrawal for either L2 token. Mintable withdrawals are
    ///         excluded because this test uses a non-mintable ERC20, which the bridge finalizes
    ///         through escrow.
    function holdTokenWithdrawal(bool _secondPair, uint256 _amount) external {
        _amount = bound(_amount, 0, 1e30);
        uint256 itemNonce = nonce++;
        address l2Token = l2Tokens[_secondPair ? 1 : 0];
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            aliased: false,
            to: l1CrossDomainMessenger,
            ethAmount: 0,
            l2Value: 0,
            gasLimit: 500_000,
            isCreation: false,
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    Encoding.encodeVersionedNonce(uint240(itemNonce), 1),
                    Predeploys.L2_STANDARD_BRIDGE,
                    address(bridge),
                    0,
                    200_000,
                    abi.encodeCall(
                        IStandardBridge.finalizeBridgeERC20,
                        (l1Token, l2Token, address(0xA11CE), address(0xB0B), _amount, hex"")
                    )
                )
            ),
            nonce: itemNonce
        });
        _hold(item, BridgeHookTypes.TokenKind.Escrowed, _amount);
    }

    /// @notice Records a verdict as the current policy for a previously held item, including a
    ///         completed item.
    function recordVerdict(uint256 _index, uint64 _expiresAt) external {
        if (ids.length == 0) return;
        _index = bound(_index, 0, ids.length - 1);
        _expiresAt = uint64(bound(_expiresAt, 1, type(uint64).max));
        vm.prank(address(module.policy()));
        module.recordVerdict(ids[_index], _expiresAt);
        verdictIndexes.push(_index);
    }

    /// @notice Revokes a verdict as the current policy for a previously held item, including a
    ///         completed item.
    function revokeVerdict(uint256 _index) external {
        if (ids.length == 0) return;
        vm.prank(address(module.policy()));
        module.revokeVerdict(ids[bound(_index, 0, ids.length - 1)]);
    }

    /// @notice Attempts completion from _caller for an item that previously received a verdict.
    ///         Mocks the policy's release decision, records unexpected success or failure, and
    ///         counts Completed events.
    function complete(uint256 _index, bool _approve, address _caller) external {
        if (verdictIndexes.length == 0) return;
        BridgeHookTypes.Item memory item = heldItems[verdictIndexes[bound(_index, 0, verdictIndexes.length - 1)]];
        IComplianceModule.Record memory record = module.items(BridgeHookHashing.hashItem(item));
        bool expected = record.heldAt != 0 && record.expiresAt != 0 && _approve;
        vm.mockCall(address(module.policy()), abi.encodePacked(IPolicy.screenRelease.selector), abi.encode(_approve));

        bool succeeded;
        vm.recordLogs();
        vm.prank(_caller);
        if (item.direction == BridgeHookTypes.Direction.Deposit) {
            try module.completeDeposit(item) {
                succeeded = true;
            } catch { }
        } else {
            try module.completeWithdrawal(item) {
                succeeded = true;
            } catch { }
        }
        if (succeeded != expected) completionMismatch = true;
        Vm.Log[] memory logs = vm.getRecordedLogs();
        for (uint256 i; i < logs.length; i++) {
            if (logs[i].emitter == address(module) && logs[i].topics[0] == keccak256("Completed(bytes32)")) {
                completedCount[logs[i].topics[1]]++;
            }
        }
    }

    /// @notice Sets one of three policies as the owner.
    function setPolicy(uint256 _index) external {
        vm.prank(owner);
        module.setPolicy(policies[bound(_index, 0, 2)]);
    }

    /// @notice Disables screening as the owner.
    function disable(uint256 _cutoffL2Nonce) external {
        vm.prank(owner);
        module.disable(_cutoffL2Nonce);
    }

    /// @notice Enables screening as the owner.
    function enable() external {
        vm.prank(owner);
        module.enable();
    }

    /// @notice Sets Portal and bridge storage to simulate a hold, funds the required balances,
    ///         and calls the module's hold function as the Portal.
    function _hold(BridgeHookTypes.Item memory _item, BridgeHookTypes.TokenKind _tokenKind, uint256 _amount) internal {
        bytes32 id = BridgeHookHashing.hashItem(_item);
        bool withdrawal = _item.direction == BridgeHookTypes.Direction.Withdrawal;

        uint256 heldSlot = withdrawal ? heldWithdrawalsSlot : heldDepositsSlot;
        vm.store(address(portal), keccak256(abi.encode(id, heldSlot)), bytes32(uint256(1)));
        vm.store(address(bridge), keccak256(abi.encode(id, heldTokenKindsSlot)), bytes32(uint256(_tokenKind)));
        if (_tokenKind == BridgeHookTypes.TokenKind.Escrowed) {
            deal(l1Token, address(module), IERC20(l1Token).balanceOf(address(module)) + _amount);
            escrowedAmountOf[id] = _amount;
        }
        vm.deal(address(portal), address(portal).balance + _item.ethAmount);

        vm.prank(address(portal));
        if (withdrawal) module.holdWithdrawal{ value: _item.ethAmount }(_item);
        else module.holdDeposit{ value: _item.ethAmount }(_item);

        heldItems.push(_item);
        ids.push(id);
        ethAmountOf[id] = _item.ethAmount;
    }
}

/// @title ComplianceModule_Invariant
/// @notice Invariants of the module's held value and completions.
contract ComplianceModule_Invariant is ComplianceModule_TestInit {
    ComplianceModule_User internal actor;

    function setUp() public override {
        super.setUp();

        actor = new ComplianceModule_User(
            vm,
            module,
            optimismPortal2,
            l1StandardBridge,
            address(l1CrossDomainMessenger),
            address(L1Token),
            [address(L2Token), address(LegacyL2Token)],
            moduleOwner,
            [policy, IPolicy(makeAddr("secondPolicy")), IPolicy(makeAddr("thirdPolicy"))]
        );

        targetContract(address(actor));

        bytes4[] memory selectors = new bytes4[](9);
        selectors[0] = actor.holdEth.selector;
        selectors[1] = actor.holdTokenDeposit.selector;
        selectors[2] = actor.holdTokenWithdrawal.selector;
        selectors[3] = actor.recordVerdict.selector;
        selectors[4] = actor.revokeVerdict.selector;
        selectors[5] = actor.complete.selector;
        selectors[6] = actor.setPolicy.selector;
        selectors[7] = actor.disable.selector;
        selectors[8] = actor.enable.selector;
        targetSelector(FuzzSelector({ addr: address(actor), selectors: selectors }));
    }

    /// @custom:invariant The module's ETH balance equals the total ETH in items still held.
    ///                   Its L1 test-token balance and heldTokensTotal entry each equal the total
    ///                   escrowed amount in items still held across both L2 token pairs.
    function invariant_heldValue_matchesOpenItems() external view {
        uint256 openEth;
        uint256 openEscrowed;
        for (uint256 i; i < actor.idCount(); i++) {
            bytes32 id = actor.ids(i);
            if (module.items(id).heldAt == 0) continue;
            openEth += actor.ethAmountOf(id);
            openEscrowed += actor.escrowedAmountOf(id);
        }

        assertEq(address(module).balance, openEth);
        assertEq(module.heldTokensTotal(address(L1Token)), openEscrowed);
        assertEq(L1Token.balanceOf(address(module)), module.heldTokensTotal(address(L1Token)));
    }

    /// @custom:invariant In this test setup, each completion attempt succeeds if and only if the
    ///                   item is held, has a recorded clear verdict, and the policy approves
    ///                   release.
    function invariant_complete_matchesRecordAndApproval() external view {
        assertFalse(actor.completionMismatch());
    }

    /// @custom:invariant The module emits Completed at most once per item.
    function invariant_completed_atMostOnce() external view {
        for (uint256 i; i < actor.idCount(); i++) {
            assertLe(actor.completedCount(actor.ids(i)), 1);
        }
    }
}
