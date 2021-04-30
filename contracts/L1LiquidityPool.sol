// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

import "./ERC20.sol";
import { L2LiquidityPool } from "./L2LiquidityPool.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "enyalabs_contracts/build/contracts/libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @dev An L1 LiquidityPool implementation
 */
contract L1LiquidityPool is OVM_CrossDomainEnabled {
    /*************
     * Variables *
     *************/

    mapping(address => uint256) balances;
    mapping(address => uint256) fees;
    
    address owner;
    address l2LiquidityPoolAddress;
    address l2ETHAddress;
    uint256 fee;

    /********************
     *    Constructor   *
     ********************/
    constructor(
        address _l2LiquidityPoolAddress,
        address _l1messenger,
        address _l2ETHAddress,
        uint256 _fee
    )
        OVM_CrossDomainEnabled(_l1messenger)
    {
        l2LiquidityPoolAddress = _l2LiquidityPoolAddress;
        l2ETHAddress = _l2ETHAddress;
        owner = msg.sender;
        fee = _fee;
    }

    /********************
     *       Events     *
     ********************/

    event ownerAddERC20Liquidity_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    event ownerRecoverFee_EVENT(
        address sender,
        address receiver,
        address erc20ContractAddress,
        uint256 amount
    );

    event clientDepositL1_EVENT(
        address sender,
        uint256 amount,
        uint256 fee,
        address erc20ContractL1Address,
        address erc20ContractL2Address
    );

    event clientPayL1_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyOwner() {
        require(msg.sender == owner, "Only the owner can call this function.");
        _;
    }

    /********************
     * Public Functions *
     ********************/

    // Default gas value which can be overridden if more complex logic runs on L2.
    uint32 public DEFAULT_FINALIZE_DEPOSIT_L2_GAS = 1200000;

    /**
     * @dev Update the transaction fee
     *
     * @param _fee Transaction fee
     */
    function init(
        uint256 _fee
    )
        public
        onlyOwner()
    {
        fee = _fee;
    }

    /**
     * @dev Receive ETH
     *
     */
    receive() external payable {

        if (msg.sender != owner) {
            uint256 _swapFee = msg.value * fee / 100; //dangerous
            uint256 _receivedAmount = msg.value - _swapFee; //needs check

            fees[address(0)] += _swapFee;

            // Construct calldata for L2LiquidityPool.depositToFinalize(_to, _amount)
            bytes memory data = abi.encodeWithSelector(
                L2LiquidityPool.clientPayL2.selector,
                msg.sender,
                _receivedAmount,
                l2ETHAddress
            );

            // Send calldata into L2
            sendCrossDomainMessage(
                l2LiquidityPoolAddress,
                data,
                getFinalizeDepositL2Gas()
            );
        }

        balances[address(0)] += msg.value;
    }

    /**
     * @dev Overridable getter for the L2 gas limit, in the case it may be
     * dynamic, and the above public constant does not suffice.
     *
     */
    function getFinalizeDepositL2Gas()
        internal
        view
        returns(
            uint32
        )
    {
        return DEFAULT_FINALIZE_DEPOSIT_L2_GAS;
    }

    /**
     * Get the fee ratio.
     * @return the fee ratio.
     */
    function getFeeRatio()
        external
        view
        returns(
            uint256
        )
    {
        return fee;
    }

    /**
     * Checks the balance of an address.
     * @param _erc20ContractAddress Address of ERC20
     * @return Balance of the address.
     */
    function balanceOf(
        address _erc20ContractAddress
    )
        external
        view
        returns (
            uint256
        )
    {
        return balances[_erc20ContractAddress];
    }

    /**
     * Checks the fee balance of an address.
     * @param _erc20ContractAddress Address of ERC20.
     * @return Balance of the address.
     */
    function feeBalanceOf(
        address _erc20ContractAddress
    )
        external
        view
        returns (
            uint256
        )
    {
        return fees[_erc20ContractAddress];
    }

    /**
     * Add ERC20 to pool
     * @param _amount Amount to be transferred into this account.
     * @param _erc20L1ContractAddress ERC20 L1 token address.
     */
    function ownerAddERC20Liquidity(
        uint256 _amount,
        address _erc20L1ContractAddress
    ) 
        external
        onlyOwner() 
    {
        ERC20 erc20Contract = ERC20(_erc20L1ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        require(erc20Contract.transferFrom(msg.sender, address(this), _amount), "ERC20 token transfer was unsuccessful");

        balances[_erc20L1ContractAddress] += _amount;

        emit ownerAddERC20Liquidity_EVENT(
            msg.sender,
            _amount,
            _erc20L1ContractAddress
        );
    }

    /**
     * Client deposit ERC20 from their account to this contract, which then releases funds on the L2 side
     * @param _amount Amount to transfer to the other account.
     * @param _erc20L1ContractAddress ERC20 L1 token address.
     * @param _erc20L2ContractAddress ERC20 L2 token address.
     */
    function clientDepositL1(
        uint256 _amount,
        address _erc20L1ContractAddress,
        address _erc20L2ContractAddress
    )
        external
    {
        ERC20 erc20Contract = ERC20(_erc20L1ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        require(erc20Contract.transferFrom(msg.sender, address(this), _amount), "ERC20 token transfer was unsuccessful");
        
        //Augment the pool size for this ERC20
        uint256 _swapFee = _amount * fee / 100; //dangerous
        uint256 _receivedAmount = _amount - _swapFee; //needs check
        balances[_erc20L1ContractAddress] += _amount;
        fees[_erc20L1ContractAddress] += _swapFee;

        // Construct calldata for L2LiquidityPool.depositToFinalize(_to, _receivedAmount)
        bytes memory data = abi.encodeWithSelector(
            L2LiquidityPool.clientPayL2.selector,
            msg.sender,
            _receivedAmount,
            _erc20L2ContractAddress
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2LiquidityPoolAddress,
            data,
            getFinalizeDepositL2Gas()
        );

        emit clientDepositL1_EVENT(
            msg.sender,
            _receivedAmount,
            _swapFee,
            _erc20L1ContractAddress,
            _erc20L2ContractAddress
        );

    }

    /**
     * owner recover fee from ERC20
     * @param _amount Amount to transfer to the other account.
     * @param _erc20ContractAddress ERC20 token address.
     * @param _to receiver to get the fee.
     */
    function ownerRecoverFee(
        uint _amount,
        address _erc20ContractAddress,
        address payable _to
    )
        external
        onlyOwner()
    {   
        if (_erc20ContractAddress != address(0)) {
            //we are dealing with an ERC20
            ERC20 erc20Contract = ERC20(_erc20ContractAddress);
            require(fees[_erc20ContractAddress] >= _amount);
            require(erc20Contract.balanceOf(address(this)) >= _amount);
            require(erc20Contract.transfer(_to, _amount));
            fees[_erc20ContractAddress] -= _amount;
        } else {
            //we are dealing with Ether
            //address(this).balance is not supported
            require(fees[address(0)] >= _amount);
            //_to.transfer(_amount); //unsafe
            // Call returns a boolean value indicating success or failure.
            // This is the current recommended method to use.
            //(bool sent,) = _to.call{value: msg.value}("");
            (bool sent,) = _to.call{value: _amount}("");
            require(sent, "Failed to send Ether");
            fees[address(0)] -= _amount;
        }

        emit ownerRecoverFee_EVENT(
            msg.sender, //which is == owner, otherwise would not have gotten here
            _to,
            _erc20ContractAddress,
            _amount
        );
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * Move funds from L2 to L1, and pay out from the right liquidity pool
     * @param _to Address that will receive the funds.
     * @param _amount amount to be transferred.
     * @param _erc20ContractAddress L1 erc20 token.
     */
    function clientPayL1(
        address payable _to,
        uint256 _amount,
        address _erc20ContractAddress
    )
        external
        onlyFromCrossDomainAccount(address(l2LiquidityPoolAddress))
    {   
        if (_erc20ContractAddress != address(0)) {
            //dealing with an ERC20
            ERC20 erc20Contract = ERC20(_erc20ContractAddress);
            require(erc20Contract.transfer(_to, _amount));
            balances[_erc20ContractAddress] -= _amount;
        } else {
            //this is ETH
            //_to.transfer(_amount); UNSAFE
            (bool sent,) = _to.call{value: _amount}("");
            require(sent, "Failed to send Ether");
            balances[address(0)] -= _amount;
        }
        
        emit clientPayL1_EVENT(
          _to,
          _amount,
          _erc20ContractAddress
        );
    }
}
