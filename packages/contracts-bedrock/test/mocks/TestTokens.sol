// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

/// @title TestERC20
/// @notice Simple ERC20 used as the original token in SuperchainERC20Factory tests.
contract TestERC20 {
    string public name;
    string public symbol;
    uint8 public decimals;
    uint256 public totalSupply;
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    constructor(string memory _name, string memory _symbol, uint8 _decimals) {
        name = _name;
        symbol = _symbol;
        decimals = _decimals;
    }

    function mint(address _to, uint256 _amount) external {
        balanceOf[_to] += _amount;
        totalSupply += _amount;
    }

    function approve(address _spender, uint256 _amount) external returns (bool) {
        allowance[msg.sender][_spender] = _amount;
        return true;
    }

    function transfer(address _to, uint256 _amount) public virtual returns (bool) {
        balanceOf[msg.sender] -= _amount;
        balanceOf[_to] += _amount;
        return true;
    }

    function transferFrom(address _from, address _to, uint256 _amount) public virtual returns (bool) {
        uint256 allowed = allowance[_from][msg.sender];
        if (allowed != type(uint256).max) allowance[_from][msg.sender] = allowed - _amount;
        balanceOf[_from] -= _amount;
        balanceOf[_to] += _amount;
        return true;
    }
}

/// @title FeeOnTransferERC20
/// @notice ERC20 that takes a 1% fee on every transferFrom, used to test that wrapping mints only
///         the received amount.
contract FeeOnTransferERC20 is TestERC20 {
    constructor() TestERC20("Fee Token", "FEE", 18) { }

    function transferFrom(address _from, address _to, uint256 _amount) public override returns (bool) {
        uint256 fee = _amount / 100;
        super.transferFrom(_from, _to, _amount - fee);
        balanceOf[_from] -= fee;
        totalSupply -= fee;
        return true;
    }
}

/// @title NoMetadataERC20
/// @notice Minimal token without the ERC20 metadata extension, used to test the metadata fallback.
contract NoMetadataERC20 {
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    function mint(address _to, uint256 _amount) external {
        balanceOf[_to] += _amount;
    }

    function approve(address _spender, uint256 _amount) external returns (bool) {
        allowance[msg.sender][_spender] = _amount;
        return true;
    }

    function transfer(address _to, uint256 _amount) external returns (bool) {
        balanceOf[msg.sender] -= _amount;
        balanceOf[_to] += _amount;
        return true;
    }

    function transferFrom(address _from, address _to, uint256 _amount) external returns (bool) {
        allowance[_from][msg.sender] -= _amount;
        balanceOf[_from] -= _amount;
        balanceOf[_to] += _amount;
        return true;
    }
}
