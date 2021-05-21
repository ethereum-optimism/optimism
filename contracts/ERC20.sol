// SPDX-License-Identifier: MIT
pragma solidity >=0.6.0 <0.8.0;
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/**
 * @title ERC20
 * @dev A super simple ERC20 implementation!
 */
contract ERC20 is IERC20 {
    /*************
     * Variables *
     *************/
    uint256 constant private MAX_UINT256 = 2**256 - 1;
    mapping (address => uint256) public balances;
    mapping (address => mapping (address => uint256)) public allowances;

    // Some optional extra goodies
    string public name;
    uint8 public decimals;
    string public symbol;                 //An identifier: eg SBX
    uint256 public override totalSupply;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _initialSupply Initial maximum token supply.
     * @param _name A name for our ERC20 (technically optional, but it's fun ok jeez).
     */
    constructor(
        uint256 _initialSupply,
        string memory _name,
        uint8 _decimalUnits,
        string memory _tokenSymbol
    ) {
        balances[msg.sender] = _initialSupply;
        totalSupply = _initialSupply;
        name = _name;
        decimals = _decimalUnits;
        symbol = _tokenSymbol;
    }

    /********************
     * Public Functions *
     ********************/

     /**
      * Mints new coins to the sender.
      * @param _amount  Amount to mint.
      * @return true if the mint was successful.
      */
     function mint(
         uint256 _amount
     )
         public
         returns (
             bool
         )
     {
         //TODO SafeMath here
         balances[msg.sender] += _amount;
         totalSupply += _amount;
         return true;
     }

    /**
     * Checks the balance of an address.
     * @param _owner Address to check a balance for.
     * @return Balance of the address.
     */
    function balanceOf(
        address _owner
    )
        external
        override
        view
        returns (
            uint256
        )
    {
        return balances[_owner];
    }

    /**
     * Transfers a balance from your account to someone else's account!
     * @param _to Address to transfer a balance to.
     * @param _amount Amount to transfer to the other account.
     * @return true if the transfer was successful.
     */
    function transfer(
        address _to,
        uint256 _amount
    )
        external
        override
        returns (
            bool
        )
    {
        require(
            balances[msg.sender] >= _amount,
            "Cannot Transfer: Insufficent Balance"
        );

        balances[msg.sender] -= _amount;
        balances[_to] += _amount;

        emit Transfer(
            msg.sender,
            _to,
            _amount
        );

        return true;
    }

    /**
     * Transfers a balance from someone else's account to another account. You need an allowance
     * from the sending account for this to work!
     * @param _from Account to transfer a balance from.
     * @param _to Account to transfer a balance to.
     * @param _amount Amount to transfer to the other account.
     * @return true if the transfer was successful.
     */
    function transferFrom(
        address _from,
        address _to,
        uint256 _amount
    )
        external
        override
        returns (
            bool
        )
    {

        uint256 allowAmount = allowances[_from][msg.sender];

        require(
            balances[_from] >= _amount,
            "Cannot TransferFrom: Balance too small."
        );

        require(
            allowAmount >= _amount,
            "Cannot TransferFrom: Allowance too small."
        );

        balances[_to] += _amount;
        balances[_from] -= _amount;

        if (allowAmount < MAX_UINT256) {
            allowances[_from][msg.sender] -= _amount;
        }

        emit Transfer(
            _from,
            _to,
            _amount
        );

        return true;
    }

    /**
     * Approves an account to spend some amount from your account.
     * @param _spender Account to approve a balance for.
     * @param _amount Amount to allow the account to spend from your account.
     * @return true if the allowance was successful.
     */
    function approve(
        address _spender,
        uint256 _amount
    )
        external
        override
        returns (
            bool
        )
    {
        allowances[msg.sender][_spender] = _amount;

        emit Approval(
            msg.sender,
            _spender,
            _amount
        );

        return true;
    }

    /**
     * Checks how much a given account is allowed to spend from another given account.
     * @param _owner Address of the account to check an allowance from.
     * @param _spender Address of the account trying to spend from the owner.
     * @return Allowance for the spender from the owner.
     */
    function allowance(
        address _owner,
        address _spender
    )
        external
        override
        view
        returns (
            uint256
        )
    {
        return allowances[_owner][_spender];
    }

    /**
     * Mints new coins to an account.
     * @param _owner Address of the account to mint to.
     * @param _amount  Amount to mint.
     * @return true if the mint was successful.
     */
    function _mint(
        address _owner,
        uint256 _amount
    )
        internal
        returns (
            bool
        )
    {
        //TODO SafeMath here
        balances[_owner] += _amount;
        totalSupply += _amount;
        return true;
    }

    /**
     * Burns coins from an account.
     * @param _owner Address of the account to mint to.
     * @param _amount  Amount to mint.
     * @return true if the mint was successful.
     */
    function _burn(
        address _owner,
        uint256 _amount
    )
        internal
        returns (
            bool
        )
    {
        require(balances[_owner] >= _amount, "Account doesn't have enough coins to burn");
        balances[_owner] -= _amount;
        totalSupply -= _amount; //TODO SafeMath here
        return true;
    }
}
