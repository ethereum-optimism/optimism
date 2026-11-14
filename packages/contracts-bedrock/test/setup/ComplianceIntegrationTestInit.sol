// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Vm } from "forge-std/Vm.sol";
import { OptimismPortal2_TestInit } from "test/setup/OptimismPortal2TestInit.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Contracts
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

// Libraries
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Features } from "src/libraries/Features.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";
import { GameStatus, GameType } from "src/dispute/lib/Types.sol";

// Interfaces
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";

/// @title ComplianceIntegration_TestInit
/// @notice Deploys the ComplianceModule and VisionPolicyV1 next to the full OP Stack deployment
///         and sets the module as the Portal's bridge hook through an OPCM upgrade. Nothing is
///         mocked except the dispute game's root claim, which stands in for the L2 state.
///
///         Deposits are sent through the real entry points. Their items are read back from the
///         module's Held event, as a completer does. Withdrawals are built by the helpers below,
///         proven against the Portal and finalized through it.
abstract contract ComplianceIntegration_TestInit is OptimismPortal2_TestInit {
    /// @notice Emitted by the module when it holds an item.
    event Held(bytes32 indexed id, BridgeHookTypes.Item item);

    /// @notice Topic of the Portal's TransactionDeposited event.
    bytes32 internal constant TRANSACTION_DEPOSITED = keccak256("TransactionDeposited(address,address,uint256,bytes)");

    uint256 internal constant AMOUNT = 1 ether;
    uint64 internal constant HOLD_DELAY = 1 hours;
    uint64 internal constant MAX_VERDICT_VALIDITY = 3 days;
    uint64 internal constant VERDICT_VALIDITY = 1 days;
    uint64 internal constant DEPOSIT_GAS_LIMIT = 100_000;
    uint32 internal constant MESSAGE_GAS_LIMIT = 200_000;

    /// @notice Owner of the module's ProxyAdmin, and so custodian of held value.
    address internal custodian = makeAddr("custodian");
    address internal moduleOwner = makeAddr("moduleOwner");
    address internal policyOwner = makeAddr("policyOwner");
    address internal verdictWriter = makeAddr("verdictWriter");
    /// @notice An address with no role, used for permissionless calls.
    address internal anyone = makeAddr("anyone");

    IProxyAdmin internal moduleProxyAdmin;
    IComplianceModule internal module;
    IVisionPolicyV1 internal policy;

    function setUp() public virtual override {
        skipIfDevFeatureDisabled(DevFeatures.BRIDGE_HOOK);
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        skipIfForkTest("ComplianceIntegration_TestInit: upgrades the chain from its local deploy config");

        _deployCompliance();
        // The upgrade runs before the dispute game used for proofs is created.
        _upgradeChain(address(module));
        super.setUp();

        vm.recordLogs();
    }

    ////////////////////////////////////////////////////////////////
    //                    Deployment and upgrades                 //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploys the module behind a proxy and the policy that serves it. The policy needs
    ///         the module's address and the module's initializer needs the policy's, so the proxy
    ///         is created first and initialized last.
    function _deployCompliance() internal {
        moduleProxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (custodian)))
            })
        );
        module = IComplianceModule(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IProxy.__constructor__, (address(moduleProxyAdmin)))
                )
            })
        );
        policy = _deployPolicy(policyOwner, verdictWriter);
        address impl = _deployModuleImpl();

        vm.prank(custodian);
        moduleProxyAdmin.upgradeAndCall(
            payable(address(module)),
            impl,
            abi.encodeCall(
                IComplianceModule.initialize,
                (moduleOwner, IPolicy(address(policy)), optimismPortal2, l1CrossDomainMessenger, l1StandardBridge)
            )
        );
    }

    /// @notice Deploys a module implementation.
    function _deployModuleImpl() internal returns (address impl_) {
        impl_ = DeployUtils.create1({
            _name: "ComplianceModule",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IComplianceModule.__constructor__, ()))
        });
    }

    /// @notice Deploys a policy for the module with empty lists.
    function _deployPolicy(address _owner, address _verdictWriter) internal returns (IVisionPolicyV1 policy_) {
        policy_ = IVisionPolicyV1(
            DeployUtils.create1({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IVisionPolicyV1.__constructor__,
                        (
                            module,
                            _owner,
                            _verdictWriter,
                            MAX_VERDICT_VALIDITY,
                            HOLD_DELAY,
                            new address[](0),
                            new address[](0)
                        )
                    )
                )
            })
        );
    }

    /// @notice Upgrades the chain through OPCM as its ProxyAdmin owner. A nonzero `_hook` is set
    ///         as the Portal's bridge hook. The zero address keeps the current hook.
    ///
    ///         The test deployment respects a game type that its OPCM config does not enable, so
    ///         the upgrade runs with the config's game type and the test one is restored after.
    function _upgradeChain(address _hook) internal {
        IOPContractsManagerV2.FullConfig memory cfg = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION)
            ? deploy.getSuperRootDeployInputV2()
            : deploy.getDeployInputV2();
        GameType testGameType = anchorStateRegistry.respectedGameType();

        IOPContractsManagerV2.UpgradeInput memory input;
        input.systemConfig = systemConfig;
        input.disputeGameConfigs = cfg.disputeGameConfigs;
        input.extraInstructions = new IOPContractsManagerUtils.ExtraInstruction[](_hook == address(0) ? 1 : 2);
        input.extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({
            key: "overrides.cfg.startingRespectedGameType", data: abi.encode(cfg.startingRespectedGameType)
        });
        if (_hook != address(0)) {
            input.extraInstructions[1] = IOPContractsManagerUtils.ExtraInstruction({
                key: "overrides.cfg.bridgeHook", data: abi.encode(IBridgeHook(_hook))
            });
        }

        prankDelegateCall(proxyAdmin.owner());
        (bool success,) = address(opcmV2).delegatecall(abi.encodeCall(IOPContractsManagerV2.upgrade, (input)));
        assertTrue(success, "OPCM upgrade failed");

        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(testGameType);
    }

    ////////////////////////////////////////////////////////////////
    //                        Policy actions                      //
    ////////////////////////////////////////////////////////////////

    function _allowlist(address _account) internal {
        vm.prank(policyOwner);
        policy.addToAllowlist(_one(_account));
    }

    function _deny(address _account) internal {
        vm.prank(policyOwner);
        policy.addToDenylist(_one(_account));
    }

    function _undeny(address _account) internal {
        vm.prank(policyOwner);
        policy.removeFromDenylist(_one(_account));
    }

    /// @notice Records a clear verdict for an item as the verdict writer.
    function _clear(bytes32 _id) internal {
        vm.prank(verdictWriter);
        policy.clear(_id, _expiry());
    }

    /// @notice The expiry the tests give every verdict.
    function _expiry() internal view returns (uint64) {
        return uint64(block.timestamp) + VERDICT_VALIDITY;
    }

    function _one(address _account) internal pure returns (address[] memory accounts_) {
        accounts_ = new address[](1);
        accounts_[0] = _account;
    }

    ////////////////////////////////////////////////////////////////
    //                           Deposits                         //
    ////////////////////////////////////////////////////////////////

    /// @notice Deposits ETH straight into the Portal. A sender without code deposits as an EOA.
    ///         A sender with code deposits as a contract, so the Portal aliases it.
    function _depositETH(address _from, address _to, uint256 _value) internal {
        vm.getRecordedLogs();
        vm.deal(_from, _from.balance + _value);
        if (_from.code.length == 0) vm.prank(_from, _from);
        else vm.prank(_from);
        optimismPortal2.depositTransaction{ value: _value }(_to, _value, DEPOSIT_GAS_LIMIT, false, hex"");
    }

    /// @notice Deposits ETH through the L1StandardBridge.
    function _depositBridgeETH(address _from, address _to, uint256 _value) internal {
        vm.getRecordedLogs();
        vm.deal(_from, _from.balance + _value);
        vm.prank(_from, _from);
        l1StandardBridge.bridgeETHTo{ value: _value }(_to, MESSAGE_GAS_LIMIT, hex"");
    }

    /// @notice Deposits L1Token, which the bridge escrows, through the L1StandardBridge.
    function _depositERC20(address _from, address _to, uint256 _amount) internal {
        _depositToken(address(L1Token), address(L2Token), _from, _to, _amount);
    }

    /// @notice Deposits a token pair through the L1StandardBridge.
    function _depositToken(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount) internal {
        vm.getRecordedLogs();
        deal(_l1Token, _from, _amount, true);
        vm.prank(_from);
        ERC20(_l1Token).approve(address(l1StandardBridge), _amount);
        vm.prank(_from, _from);
        l1StandardBridge.bridgeERC20To(_l1Token, _l2Token, _to, _amount, MESSAGE_GAS_LIMIT, hex"");
    }

    /// @notice Sends a message that carries no value through the L1CrossDomainMessenger.
    function _depositMessage(address _from, address _target, bytes memory _message) internal {
        vm.getRecordedLogs();
        vm.prank(_from, _from);
        l1CrossDomainMessenger.sendMessage(_target, _message, MESSAGE_GAS_LIMIT);
    }

    /// @notice Returns the item of the deposit held since the last deposit, as the module emitted
    ///         it. Asserts that exactly one item was held and that no deposit was published.
    ///         Reading the logs consumes them.
    function _heldDeposit() internal view returns (bytes32 id_, BridgeHookTypes.Item memory item_) {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        (uint256 held, Vm.Log memory log) = _find(logs, address(module), Held.selector);
        (uint256 published,) = _find(logs, address(optimismPortal2), TRANSACTION_DEPOSITED);
        assertEq(held, 1, "items held");
        assertEq(published, 0, "deposits published by a hold");
        id_ = log.topics[1];
        item_ = abi.decode(log.data, (BridgeHookTypes.Item));
    }

    /// @notice Returns a hash of the deposit the Portal published since the last deposit or log
    ///         read. Two deposits with the same hash are the same L2 transaction. Asserts that
    ///         exactly one deposit was published. Reading the logs consumes them.
    function _publishedDeposit() internal view returns (bytes32) {
        (uint256 published, Vm.Log memory log) =
            _find(vm.getRecordedLogs(), address(optimismPortal2), TRANSACTION_DEPOSITED);
        assertEq(published, 1, "deposits published");
        return keccak256(abi.encode(log.topics, log.data));
    }

    /// @notice Completes a held deposit as an address with no role. Asserts that the Portal
    ///         published exactly one deposit, with the terms of the held item, and returns its
    ///         hash.
    function _completeDeposit(BridgeHookTypes.Item memory _item) internal returns (bytes32 published_) {
        vm.getRecordedLogs();
        vm.prank(anyone);
        module.completeDeposit(_item);
        published_ = _publishedDeposit();

        bytes32[] memory topics = new bytes32[](4);
        topics[0] = TRANSACTION_DEPOSITED;
        topics[1] = bytes32(uint256(uint160(_item.from)));
        topics[2] = bytes32(uint256(uint160(_item.to)));
        bytes memory opaqueData =
            abi.encodePacked(_item.ethAmount, _item.l2Value, uint64(_item.gasLimit), _item.isCreation, _item.data);
        assertEq(published_, keccak256(abi.encode(topics, abi.encode(opaqueData))), "published deposit");
    }

    /// @notice Returns the number of deposits the Portal published since the last deposit or log
    ///         read. Reading the logs consumes them.
    function _publishedDepositCount() internal view returns (uint256 count_) {
        (count_,) = _find(vm.getRecordedLogs(), address(optimismPortal2), TRANSACTION_DEPOSITED);
    }

    /// @notice Counts the logs with the given emitter and first topic, and returns the last one.
    function _find(
        Vm.Log[] memory _logs,
        address _emitter,
        bytes32 _topic
    )
        private
        pure
        returns (uint256 count_, Vm.Log memory last_)
    {
        for (uint256 i; i < _logs.length; i++) {
            if (_logs[i].emitter == _emitter && _logs[i].topics[0] == _topic) {
                last_ = _logs[i];
                count_++;
            }
        }
    }

    ////////////////////////////////////////////////////////////////
    //                          Withdrawals                       //
    ////////////////////////////////////////////////////////////////

    /// @notice Builds a withdrawal sent straight to the L2ToL1MessagePasser.
    function _withdrawal(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value
    )
        internal
        pure
        returns (Types.WithdrawalTransaction memory)
    {
        return Types.WithdrawalTransaction({
            nonce: Encoding.encodeVersionedNonce(uint240(_nonce), 1),
            sender: _sender,
            target: _target,
            value: _value,
            gasLimit: 100_000,
            data: hex""
        });
    }

    /// @notice Builds a withdrawal sent through the messengers.
    function _messageWithdrawal(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        bytes memory _message
    )
        internal
        view
        returns (Types.WithdrawalTransaction memory)
    {
        uint256 nonce = Encoding.encodeVersionedNonce(uint240(_nonce), 1);
        return Types.WithdrawalTransaction({
            nonce: nonce,
            sender: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            target: address(l1CrossDomainMessenger),
            value: _value,
            gasLimit: l1CrossDomainMessenger.baseGas(_message, MESSAGE_GAS_LIMIT),
            data: abi.encodeCall(
                ICrossDomainMessenger.relayMessage, (nonce, _sender, _target, _value, MESSAGE_GAS_LIMIT, _message)
            )
        });
    }

    /// @notice Builds a withdrawal of ETH through the standard bridges.
    function _bridgeETHWithdrawal(
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        internal
        view
        returns (Types.WithdrawalTransaction memory)
    {
        return _messageWithdrawal(
            _nonce,
            Predeploys.L2_STANDARD_BRIDGE,
            address(l1StandardBridge),
            _amount,
            abi.encodeCall(IStandardBridge.finalizeBridgeETH, (_from, _to, _amount, hex""))
        );
    }

    /// @notice Builds a withdrawal of L1Token through the standard bridges.
    function _erc20Withdrawal(
        uint256 _nonce,
        address _from,
        address _to,
        uint256 _amount
    )
        internal
        view
        returns (Types.WithdrawalTransaction memory)
    {
        return _messageWithdrawal(
            _nonce,
            Predeploys.L2_STANDARD_BRIDGE,
            address(l1StandardBridge),
            0,
            abi.encodeCall(
                IStandardBridge.finalizeBridgeERC20, (address(L1Token), address(L2Token), _from, _to, _amount, hex"")
            )
        );
    }

    /// @notice Returns the item the Portal builds for a withdrawal, so a verdict can be recorded
    ///         before the withdrawal is finalized. Use `_tokenItemOf` for a token withdrawal.
    function _itemOf(Types.WithdrawalTransaction memory _tx)
        internal
        pure
        returns (bytes32 id_, BridgeHookTypes.Item memory item_)
    {
        item_ = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: _tx.sender,
            aliased: false,
            to: _tx.target,
            ethAmount: _tx.value,
            l2Value: 0,
            gasLimit: _tx.gasLimit,
            isCreation: false,
            data: _tx.data,
            nonce: _tx.nonce
        });
        id_ = BridgeHookHashing.hashItem(item_);
    }

    /// @notice Returns the item the Portal builds for a withdrawal that it recognizes as a token
    ///         transfer between the standard bridges.
    function _tokenItemOf(Types.WithdrawalTransaction memory _tx)
        internal
        pure
        returns (bytes32 id_, BridgeHookTypes.Item memory item_)
    {
        (, item_) = _itemOf(_tx);
        item_.asset = BridgeHookTypes.Asset.ERC20;
        id_ = BridgeHookHashing.hashItem(item_);
    }

    /// @notice Finalizes a withdrawal that the policy holds, and returns its item as the module
    ///         emitted it. Asserts that exactly one item was held.
    function _holdWithdrawal(Types.WithdrawalTransaction memory _tx)
        internal
        returns (bytes32 id_, BridgeHookTypes.Item memory item_)
    {
        vm.getRecordedLogs();
        optimismPortal2.finalizeWithdrawalTransaction(_tx);
        (uint256 held, Vm.Log memory log) = _find(vm.getRecordedLogs(), address(module), Held.selector);
        assertEq(held, 1, "items held");
        id_ = log.topics[1];
        item_ = abi.decode(log.data, (BridgeHookTypes.Item));
    }

    /// @notice Proves a withdrawal and funds the ETH it pays. Several withdrawals can be proven
    ///         against the same game.
    function _prove(Types.WithdrawalTransaction memory _tx) internal returns (bytes32 withdrawalHash_) {
        vm.deal(_ethVault(), _ethVault().balance + _tx.value);

        (bytes32 stateRoot, bytes32 storageRoot, bytes32 outputRoot,, bytes[] memory withdrawalProof) =
            ffi.getProveWithdrawalTransactionInputs(_tx);
        if (DisputeGames.isSuperGame(game.gameType())) {
            vm.mockCall(
                address(game),
                abi.encodeCall(game.rootClaimByChainId, (systemConfig.l2ChainId())),
                abi.encode(outputRoot)
            );
        } else {
            vm.mockCall(address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(outputRoot));
        }

        optimismPortal2.proveWithdrawalTransaction(
            _tx,
            _proposedGameIndex,
            Types.OutputRootProof({
                version: bytes32(0),
                stateRoot: stateRoot,
                messagePasserStorageRoot: storageRoot,
                latestBlockhash: bytes32(0)
            }),
            withdrawalProof
        );
        withdrawalHash_ = Hashing.hashWithdrawal(_tx);
    }

    /// @notice Resolves the game and warps to the first second at which a withdrawal proven by
    ///         this contract can be finalized. Returns that time, which the Portal passes to the
    ///         hook as `finalizableAt`.
    function _warpToFinalizable(bytes32 _withdrawalHash) internal returns (uint256 finalizableAt_) {
        if (game.status() == GameStatus.IN_PROGRESS) {
            game.resolveClaim(0, 0);
            game.resolve();
        }

        (, uint64 provenAt) = optimismPortal2.provenWithdrawals(_withdrawalHash, address(this));
        uint256 maturedAt = provenAt + optimismPortal2.proofMaturityDelaySeconds();
        uint256 settledAt = game.resolvedAt().raw() + optimismPortal2.disputeGameFinalityDelaySeconds();
        finalizableAt_ = (maturedAt > settledAt ? maturedAt : settledAt) + 1;
        vm.warp(finalizableAt_);
    }

    /// @notice Proves a withdrawal and warps past the hold delay, where the policy holds an
    ///         uncleared withdrawal instead of rejecting it.
    function _proveAndWarpToHoldable(Types.WithdrawalTransaction memory _tx)
        internal
        returns (bytes32 withdrawalHash_)
    {
        withdrawalHash_ = _prove(_tx);
        vm.warp(_warpToFinalizable(withdrawalHash_) + HOLD_DELAY);
    }

    ////////////////////////////////////////////////////////////////
    //                          Assertions                        //
    ////////////////////////////////////////////////////////////////

    /// @notice Where the Portal keeps its ETH.
    function _ethVault() internal view returns (address) {
        return isUsingLockbox() ? address(ethLockbox) : address(optimismPortal2);
    }

    /// @notice Asserts that the Portal committed to the deposit and the module recorded its hold.
    function _assertHeldDeposit(bytes32 _id) internal view {
        assertTrue(optimismPortal2.heldDeposits(_id), "held deposit commitment");
        assertGt(module.items(_id).heldAt, 0, "module hold record");
    }

    /// @notice Asserts that the Portal committed to the withdrawal and the module recorded its hold.
    function _assertHeldWithdrawal(bytes32 _id) internal view {
        assertTrue(optimismPortal2.heldWithdrawals(_id), "held withdrawal commitment");
        assertGt(module.items(_id).heldAt, 0, "module hold record");
    }

    /// @notice Asserts that neither the Portal nor the module keeps anything for the item.
    function _assertReleased(bytes32 _id) internal view {
        assertFalse(optimismPortal2.heldDeposits(_id), "held deposit commitment");
        assertFalse(optimismPortal2.heldWithdrawals(_id), "held withdrawal commitment");
        assertEq(module.items(_id).heldAt, 0, "module hold record");
        assertEq(module.items(_id).expiresAt, 0, "module verdict");
    }

    /// @notice Asserts where L1Token sits: in the bridge's escrow or held by the module. Checks
    ///         each side's balance against its own accounting.
    function _assertTokenCustody(uint256 _escrowed, uint256 _held) internal view {
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), _escrowed, "bridge deposits");
        assertEq(L1Token.balanceOf(address(l1StandardBridge)), _escrowed, "bridge balance");
        assertEq(module.heldTokensTotal(address(L1Token)), _held, "module held total");
        assertEq(L1Token.balanceOf(address(module)), _held, "module balance");
    }
}
