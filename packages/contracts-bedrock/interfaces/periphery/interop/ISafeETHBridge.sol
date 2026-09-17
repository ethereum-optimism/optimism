// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ISafeETHBridge
/// @notice Interface for the experimental native ETH bridge.
interface ISafeETHBridge {
    enum Status { NONE, PREPARED, COMMITTED, ABORTED }
    struct Transfer {
        address sender;
        address recipient;
        uint256 amount;
        uint256 nonce;
        uint256 deadline;
    }
    struct Record { Transfer transfer; Status status; }

    error Unauthorized();
    error ZeroAddress();
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

    function __constructor__(uint256 _chainA, uint256 _chainB) external;
    function version() external view returns (string memory);
    function remoteChainId() external view returns (uint256);
    function nonce() external view returns (uint256);
    function reservedLiquidity() external view returns (uint256);
    function source(bytes32 _id) external view returns (Record memory);
    function destination(bytes32 _id) external view returns (Record memory);
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_);
    function relayETH(address _from, address _to, uint256 _amount) external;
    function initiateTransfer(address _recipient, uint256 _deadline) external payable returns (bytes32 id_);
    function prepareDestination(bytes32 _id_, Transfer calldata _transfer) external;
    function acknowledgePrepare(bytes32 _id_) external;
    function commitDestination(bytes32 _id_) external;
    function abortTransfer(bytes32 _id_) external;
    function abortDestination(bytes32 _id_, Transfer calldata _transfer) external;
}
