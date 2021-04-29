// SPDX-License-Identifier: MIT
pragma solidity >0.6.0 <0.8.0;

/**
 * @title ERC20
 * @dev A super simple ERC20 implementation! Also *very* insecure. Do not use in prod.
 */
contract ERC20 {

    /**********
     * Events *
     **********/

    event Transfer(
        address indexed _from,
        address indexed _to,
        uint256 _value
    );

    event Approval(
        address indexed _owner,
        address indexed _spender,
        uint256 _value
    );


    /*************
     * Variables *
     *************/

    mapping (address => uint256) public balances;
    mapping (address => mapping (address => uint256)) public allowances;

    // Some optional extra goodies.
    uint256 public totalSupply;
    string public name;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _initialSupply Initial maximum token supply.
     * @param _name A name for our ERC20 (technically optional, but it's fun ok jeez).
     */
    constructor(
        uint256 _initialSupply,
        string memory _name
    )
        public
    {
        balances[msg.sender] = _initialSupply;
        totalSupply = _initialSupply;
        name = _name;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Checks the balance of an address.
     * @param _owner Address to check a balance for.
     * @return Balance of the address.
     */
    function balanceOf(
        address _owner
    )
        external
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
        returns (
            bool
        )
    {
        require(
            balances[msg.sender] >= _amount,
            "You don't have enough balance to make this transfer!"
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
        returns (
            bool
        )
    {
        require(
            balances[_from] >= _amount,
            "Can't transfer from the desired account because it doesn't have enough balance."
        );

        require(
            allowances[_from][msg.sender] >= _amount,
            "Can't transfer from the desired account because you don't have enough of an allowance."
        );

        balances[_to] += _amount;
        balances[_from] -= _amount;

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
        view
        returns (
            uint256
        )
    {
        return allowances[_owner][_spender];
    }

    /**
     * Internal minting function.
     * @param _who Address to mint tokens for.
     * @param _amount Number of tokens to mint.
     */
    function _mint(
        address _who,
        uint256 _amount
    )
        internal
    {
        totalSupply += _amount;
        balances[_who] += _amount;
        emit Transfer(address(0), _who, _amount);
    }

    /**
     * Internal burning function.
     * @param _who Address to burn tokens from.
     * @param _amount Number of tokens to burn.
     */
    function _burn(
        address _who,
        uint256 _amount
    )
        internal
    {
        require(
            totalSupply >= _amount,
            "Can't burn more than total supply."
        );

        require(
            balances[_who] >= _amount,
            "Account does not have enough to burn."
        );

        totalSupply -= _amount;
        balances[_who] -= _amount;
        emit Transfer(_who, address(0), _amount);
    }
}
