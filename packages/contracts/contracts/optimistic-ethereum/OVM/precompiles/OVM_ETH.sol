// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_ERC20 } from "../../iOVM/precompiles/iOVM_ERC20.sol";
import { iOVM_BaseCrossDomainMessenger } from "../../iOVM/bridge/iOVM_BaseCrossDomainMessenger.sol";

/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that 
 * unlike on Layer 1, Layer 2 accounts do not have a balance field.
 * 
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ETH is iOVM_ERC20, Lib_AddressResolver {

    uint256 constant private MAX_UINT256 = 2**256 - 1;
    mapping (address => uint256) public balances;
    mapping (address => mapping (address => uint256)) public allowed;
    /*
    NOTE:
    The following variables are OPTIONAL vanities. One does not have to include them.
    They allow one to customise the token contract & in no way influences the core functionality.
    Some wallets/interfaces might not even bother to look at this information.
    */
    string public name;                   //fancy name: eg OVM Coin
    uint8 public decimals;                //How many decimals to show.
    string public symbol;                 //An identifier: eg OVM
    uint256 public override totalSupply;

    constructor(
        address _libAddressManager,
        uint256 _initialAmount,
        string memory _tokenName,
        uint8 _decimalUnits,
        string memory _tokenSymbol
    )
        public
        Lib_AddressResolver(_libAddressManager)
    {
        balances[msg.sender] = _initialAmount;               // Give the creator all initial tokens
        totalSupply = _initialAmount;                        // Update total supply
        name = _tokenName;                                   // Set the name for display purposes
        decimals = _decimalUnits;                            // Amount of decimals for display purposes
        symbol = _tokenSymbol;                               // Set the symbol for display purposes
    }

    modifier onlyOVMETHBridge() {
        address bridgeOnL2 = resolve("OVM_L2ETHBridge");
        require(bridgeOnL2 != address(0), "OVM_L2ETHBridge is not yet initialized.");
        require(msg.sender == bridgeOnL2, "Only callable by OVM ETH Deposit/Withdrawal contract");
        _;
    }

    function transfer(address _to, uint256 _value) external override returns (bool success) {
        require(balances[msg.sender] >= _value);
        balances[msg.sender] -= _value;
        balances[_to] += _value;
        emit Transfer(msg.sender, _to, _value);
        return true;
    }

    function transferFrom(address _from, address _to, uint256 _value) external override returns (bool success) {
        uint256 allowance = allowed[_from][msg.sender];
        require(balances[_from] >= _value && allowance >= _value);
        balances[_to] += _value;
        balances[_from] -= _value;
        if (allowance < MAX_UINT256) {
            allowed[_from][msg.sender] -= _value;
        }
        emit Transfer(_from, _to, _value);
        return true;
    }

    function balanceOf(address _owner) external view override returns (uint256 balance) {
        return balances[_owner];
    }

    function approve(address _spender, uint256 _value) external override returns (bool success) {
        allowed[msg.sender][_spender] = _value;
        emit Approval(msg.sender, _spender, _value);
        return true;
    }

    function allowance(address _owner, address _spender) external view override returns (uint256 remaining) {
        return allowed[_owner][_spender];
    }

    function mint(address _account, uint256 _amount) external onlyOVMETHBridge returns (bool success) {
        uint256 newTotalSupply = totalSupply + _amount;
        require(newTotalSupply >= totalSupply, "SafeMath: addition overflow");
        totalSupply = newTotalSupply;
        balances[_account] += _amount;

        emit Mint(_account, _amount);
        return true;
    }

    function burn(address _account, uint256 _amount) external onlyOVMETHBridge returns (bool success) {
        require(balances[_account] >= _amount, "Unable to burn due to insufficient balance");
        balances[_account] -= _amount;
        totalSupply -= _amount;

        emit Burn(_account, _amount);
        return true;
    }
}
