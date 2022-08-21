// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

interface IGelatoTreasury {
    function userTokenBalance(address _user, address _token) external view returns (uint256);
    function depositFunds(address _receiver, address _token, uint256 _amount) external;
}

contract Gelato {
    IGelatoTreasury public immutable treasury;

    constructor(IGelatoTreasury _treasury) {
        treasury = _treasury;
    }

    function balance(address _who) external view returns (uint256) {
        return treasury.userTokenBalance(
            _who,
            // Gelato represents ETH as 0xeeeee....eeeee
            0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE
        );
    }

    function deposit(address _who, uint256 _amt) external {
        treasury.depositFunds(
            _who,
            // Gelato represents ETH as 0xeeeee....eeeee
            0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE,
            _amt
        );
    }
}
