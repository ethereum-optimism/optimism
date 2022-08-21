// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract Ethereum {
    function balance(address _who) external view returns (uint256) {
        return address(_who).balance;
    }

    function timestamp() external view returns (uint256) {
        return block.timestamp;
    }

    function transfer(address payable _who, uint256 _amt) external returns (bool) {
        return _who.send(_amt);
    }

    function transfer(address payable _who, uint256 _amt, uint256 _gas) external returns (bool) {
        (bool success, ) = _who.call{gas: _gas, value: _amt}("");
        return success;
    }
}
