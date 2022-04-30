pragma solidity ^0.8.0;

interface IWithdrawer {
    event WithdrawalInitiated(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );
    event WithdrawerBalanceBurnt(uint256 indexed amount);

    function burn() external;

    function initiateWithdrawal(
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    ) external payable;

    function nonce() external view returns (uint256);

    function withdrawals(bytes32) external view returns (bool);
}
