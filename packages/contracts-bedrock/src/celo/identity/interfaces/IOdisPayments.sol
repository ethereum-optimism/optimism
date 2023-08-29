pragma solidity ^0.5.13;

interface IOdisPayments {
    function payInCUSD(address account, uint256 value) external;
    function totalPaidCUSD(address) external view returns (uint256);
}
