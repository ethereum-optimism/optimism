// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { VmSafe } from "forge-std/Vm.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract
import { IL2ToL2CrossDomainMessenger, Identifier } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { Identifier as CrossL2InboxIdentifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ISuperchainTokenBridge } from "interfaces/L2/ISuperchainTokenBridge.sol";

/// @notice Integration test that checks that the `ExecutingMessage` event is emitted on crosschain mints.
contract ExecutingMessageEmittedTest is CommonTest {
    bytes32 internal constant SENT_MESSAGE_EVENT_SELECTOR =
        0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320;

    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    address internal immutable FOUNDRY_VM_ADDRESS = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
    address internal immutable SUPERCHAIN_TOKEN_BRIDGE = Predeploys.SUPERCHAIN_TOKEN_BRIDGE;
    IL2ToL2CrossDomainMessenger internal immutable MESSENGER =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

    address public superchainERC20 = makeAddr("SuperchainERC20");

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        {
            // TODO: Remove this block when L2Genesis includes this contract.
            vm.etch(
                Predeploys.SUPERCHAIN_TOKEN_BRIDGE,
                vm.getDeployedCode("SuperchainTokenBridge.sol:SuperchainTokenBridge")
            );
        }

        vm.etch(superchainERC20, vm.getDeployedCode("OptimismSuperchainERC20.sol:OptimismSuperchainERC20"));
    }

    /// @notice Tests that when `SuperchainERC20#crosschainMint` is called, the `ExecutingMessage` event is emitted.
    /// forge-config: default.isolate = true
    function test_superchainERC20CrosschainMint_emitsExecutingMessage_succeeds(
        Identifier memory _id,
        uint256 _nonce,
        address _sender,
        address _to,
        uint256 _amount
    )
        public
    {
        _crosschainMint_emitsExecutingMessage_succeeds(superchainERC20, _id, _nonce, _sender, _to, _amount);
    }

    /// @notice Tests that when a super token mint is relayed, the `ExecutingMessage` event is emitted.
    /// @dev Using 2 different `Identifier` and `CrossL2InboxIdentifier` structs type, when it's the same type to avoid
    ///      a compiler error due to a type mismatch.
    function _crosschainMint_emitsExecutingMessage_succeeds(
        address _token,
        Identifier memory _id,
        uint256 _nonce,
        address _sender,
        address _to,
        uint256 _amount
    )
        internal
    {
        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomain- or the foundry VM
        vm.assume(_to != address(crossL2Inbox) && _to != address(MESSENGER) && _to != FOUNDRY_VM_ADDRESS);

        // Ensure that the target is not a forge address.
        assumeNotForgeAddress(_to);

        // Construct the SentMessage payload & identifier
        _id.origin = address(MESSENGER);
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);
        bytes memory message = abi.encodeCall(ISuperchainTokenBridge.relayERC20, (_token, _sender, _to, _amount));
        bytes memory sentMessage = abi.encodePacked(
            abi.encode(SENT_MESSAGE_EVENT_SELECTOR, block.chainid, SUPERCHAIN_TOKEN_BRIDGE, _nonce), // topics
            abi.encode(_sender, message) // data
        );

        // Mock `crossDomainMessageContext` call for it to succeed
        // encode call
        vm.mockCall(
            address(MESSENGER),
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(SUPERCHAIN_TOKEN_BRIDGE, _id.chainId)
        );

        // Prepare the access list to be sent with the relay call
        bytes32 slot = crossL2Inbox.calculateChecksum(
            CrossL2InboxIdentifier(_id.origin, _id.blockNumber, _id.logIndex, _id.timestamp, _id.chainId),
            keccak256(sentMessage)
        );
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slot;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect the `ExecutingMessage` event to be properly emitted
        vm.expectEmit(address(crossL2Inbox));
        bytes32 messageHash = keccak256(sentMessage);
        emit ExecutingMessage(messageHash, _id);

        // relay the message
        vm.accessList(accessList);
        MESSENGER.relayMessage(_id, sentMessage);
    }
}
