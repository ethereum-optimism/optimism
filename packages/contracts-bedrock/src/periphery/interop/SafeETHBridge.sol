// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { SafeSend } from "src/universal/SafeSend.sol";

// Libraries
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title SafeETHBridge
/// @notice Experimental replacement for SuperchainETHBridge at its EXISTING predeploy address.
///         Source ETH stays refundable until an authenticated destination ACK. PREPARE reserves
///         capacity in ETHLiquidity without minting; COMMIT locks source ETH there and authorizes
///         destination minting. Ordinary sendETH/relayETH remain available; ordinary relays may
///         only spend unreserved capacity. Safe bridging is opt-in via initiateTransfer.
///         Safety assumes canonical Interop messages, identical peer logic, and no upgrades that
///         invalidate reservations. Liveness needs eventual relay; committed ETH is never refunded.
contract SafeETHBridge is ISemver {
    enum Status {
        NONE,
        PREPARED,
        COMMITTED,
        ABORTED
    }

    struct Transfer {
        address sender;
        address recipient;
        uint256 amount;
        uint256 nonce;
        uint256 deadline;
    }

    struct Record {
        Transfer transfer;
        Status status;
    }

    /// @notice Fixed peer for this experiment; redeployment/upgrades are outside the protocol.
    // nosemgrep: sol-safety-no-immutable-variables
    uint256 internal immutable REMOTE_CHAIN_ID;

    // Preserve the ordinary SuperchainETHBridge error selector.
    // nosemgrep: sol-style-error-format
    error InvalidCrossDomainSender();

    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);
    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    error SafeETHBridge_Unauthorized();
    error SafeETHBridge_InvalidTransfer();
    error SafeETHBridge_InvalidState();
    error SafeETHBridge_InsufficientLiquidity();

    event TransferPrepared(bytes32 indexed transferId, Transfer transfer);
    event DestinationPrepared(bytes32 indexed transferId);
    event TransferCommitted(bytes32 indexed transferId);
    event DestinationCommitted(bytes32 indexed transferId);
    event TransferAborted(bytes32 indexed transferId);
    event DestinationAborted(bytes32 indexed transferId);

    /// @notice Experimental semantic version.
    /// @custom:semver 0.1.0
    string public constant version = "0.1.0";
    uint256 public nonce;
    /// @notice ETHLiquidity capacity promised to incoming transfers but not yet minted.
    uint256 public reservedLiquidity;
    mapping(bytes32 => Record) internal _source;
    mapping(bytes32 => Record) internal _destination;

    /// @notice Pins a two-chain trust domain. Both deployments use the same constructor arguments.
    constructor(uint256 _chainA, uint256 _chainB) {
        if (_chainA == _chainB || (block.chainid != _chainA && block.chainid != _chainB)) {
            revert SafeETHBridge_InvalidTransfer();
        }
        REMOTE_CHAIN_ID = block.chainid == _chainA ? _chainB : _chainA;
    }

    /// @notice Authenticates safe-protocol callbacks, including duplicates and abort-before-prepare messages.
    modifier onlyPeer() {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert SafeETHBridge_Unauthorized();
        (address sender, uint256 sourceChain) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();
        if (sender != address(this) || sourceChain != REMOTE_CHAIN_ID) {
            revert SafeETHBridge_Unauthorized();
        }
        _;
    }

    function remoteChainId() external view returns (uint256) {
        return REMOTE_CHAIN_ID;
    }

    function source(bytes32 _id) external view returns (Record memory) {
        return _source[_id];
    }

    function destination(bytes32 _id) external view returns (Record memory) {
        return _destination[_id];
    }

    /// @notice Sends ETH to some target address on another chain.
    /// @param _to       Address to send ETH to.
    /// @param _chainId  Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent.
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_) {
        if (_to == address(0)) revert ZeroAddress();

        // NOTE: 'burn' will soon change to 'deposit'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).burn{ value: msg.value }();

        msgHash_ = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _destination: _chainId,
            _target: address(this),
            _message: abi.encodeCall(this.relayETH, (msg.sender, _to, msg.value))
        });

        emit SendETH(msg.sender, _to, msg.value, _chainId);
    }

    /// @notice Relays ETH received from another chain.
    /// @param _from       Address of the msg.sender of sendETH on the source chain.
    /// @param _to         Address to relay ETH to.
    /// @param _amount     Amount of ETH to relay.
    function relayETH(address _from, address _to, uint256 _amount) external {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 sourceChain) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != address(this)) revert InvalidCrossDomainSender();

        // Ordinary delivery cannot consume capacity promised by a safe PREPARE.
        _requireUnreservedLiquidity(_amount);

        // NOTE: 'mint' will soon change to 'withdraw'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(_amount);

        // This is a forced ETH send to the recipient, the recipient should NOT expect to be called.
        new SafeSend{ value: _amount }(payable(_to));

        emit RelayETH(_from, _to, _amount, sourceChain);
    }

    /// @notice Escrows native ETH until ACK commits it or the sender aborts after the deadline.
    function initiateTransfer(address _recipient, uint256 _deadline) external payable returns (bytes32 id_) {
        Transfer memory transfer = Transfer(msg.sender, _recipient, msg.value, nonce++, _deadline);
        _validate(transfer);
        if (_deadline <= block.timestamp) revert SafeETHBridge_InvalidTransfer();
        id_ = _transferId(block.chainid, REMOTE_CHAIN_ID, transfer);
        _source[id_] = Record(transfer, Status.PREPARED);
        _send(abi.encodeCall(this.prepareDestination, (id_, transfer)));
        emit TransferPrepared(id_, transfer);
    }

    /// @notice Reserves ETHLiquidity capacity without minting. ETHLiquidity only permits the existing
    ///         SuperchainETHBridge address to mint. Ordinary relays preserve all reservations; safe
    ///         commits consume their own. Source refunds use separate escrow, never this capacity.
    function prepareDestination(bytes32 _id_, Transfer calldata _transfer) external onlyPeer {
        _validateIncoming(_id_, _transfer);
        if (_destination[_id_].status != Status.NONE) return;
        _requireUnreservedLiquidity(_transfer.amount);
        reservedLiquidity += _transfer.amount;
        _destination[_id_] = Record(_transfer, Status.PREPARED);
        _send(abi.encodeCall(this.acknowledgePrepare, (_id_)));
        emit DestinationPrepared(_id_);
    }

    /// @notice The sole commit point. At the deadline ACK becomes stale; only abort is possible.
    ///         Locking escrow in ETHLiquidity, the decision, and sending COMMIT are atomic.
    function acknowledgePrepare(bytes32 _id_) external onlyPeer {
        Record storage record = _source[_id_];
        if (record.status == Status.NONE) revert SafeETHBridge_InvalidState();
        if (record.status != Status.PREPARED || block.timestamp >= record.transfer.deadline) return;
        record.status = Status.COMMITTED;
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).burn{ value: record.transfer.amount }();
        _send(abi.encodeCall(this.commitDestination, (_id_)));
        emit TransferCommitted(_id_);
    }

    /// @notice Releases reserved ETH exactly once. SafeSend bypasses reverting recipient code.
    function commitDestination(bytes32 _id_) external onlyPeer {
        Record storage record = _destination[_id_];
        if (record.status == Status.COMMITTED || record.status == Status.ABORTED) return;
        if (record.status != Status.PREPARED) revert SafeETHBridge_InvalidState();
        record.status = Status.COMMITTED;
        reservedLiquidity -= record.transfer.amount;
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(record.transfer.amount);
        new SafeSend{ value: record.transfer.amount }(payable(record.transfer.recipient));
        emit DestinationCommitted(_id_);
    }

    /// @notice Refunds only undecided escrow. A missing destination can never undo a commit.
    function abortTransfer(bytes32 _id_) external {
        Record storage record = _source[_id_];
        if (msg.sender != record.transfer.sender) revert SafeETHBridge_Unauthorized();
        if (record.status == Status.ABORTED) return;
        if (record.status != Status.PREPARED || block.timestamp < record.transfer.deadline) {
            revert SafeETHBridge_InvalidState();
        }
        record.status = Status.ABORTED;
        _send(abi.encodeCall(this.abortDestination, (_id_, record.transfer)));
        new SafeSend{ value: record.transfer.amount }(payable(record.transfer.sender));
        emit TransferAborted(_id_);
    }

    /// @notice Retains an abort tombstone even when PREPARE has not arrived. Only the source may
    ///         cancel a reservation; a destination timeout would allow refund-plus-payout races.
    function abortDestination(bytes32 _id_, Transfer calldata _transfer) external onlyPeer {
        _validateIncoming(_id_, _transfer);
        Status status = _destination[_id_].status;
        if (status == Status.ABORTED) return;
        if (status == Status.COMMITTED) revert SafeETHBridge_InvalidState();
        if (status == Status.PREPARED) reservedLiquidity -= _transfer.amount;
        _destination[_id_] = Record(_transfer, Status.ABORTED);
        emit DestinationAborted(_id_);
    }

    function _requireUnreservedLiquidity(uint256 _amount) internal view {
        if (Predeploys.ETH_LIQUIDITY.balance < reservedLiquidity + _amount) {
            revert SafeETHBridge_InsufficientLiquidity();
        }
    }

    function _validate(Transfer memory _transfer) internal view {
        if (
            _transfer.sender == address(0) || _transfer.recipient == address(0) || _transfer.recipient == address(this)
                || _transfer.amount == 0
        ) {
            revert SafeETHBridge_InvalidTransfer();
        }
    }

    function _validateIncoming(bytes32 _id_, Transfer memory _transfer) internal view {
        _validate(_transfer);
        if (_id_ != _transferId(REMOTE_CHAIN_ID, block.chainid, _transfer)) {
            revert SafeETHBridge_InvalidTransfer();
        }
    }

    function _transferId(
        uint256 _sourceChain,
        uint256 _destinationChain,
        Transfer memory _transfer
    )
        internal
        view
        returns (bytes32)
    {
        return keccak256(abi.encode(_sourceChain, _destinationChain, address(this), _transfer));
    }

    function _send(bytes memory _message) internal {
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage(
            REMOTE_CHAIN_ID, address(this), _message
        );
    }
}
