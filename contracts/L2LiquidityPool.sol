// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

import "./ERC20.sol";
import { L1LiquidityPool } from "./L1LiquidityPool.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "enyalabs_contracts/build/contracts/libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @dev An L2 LiquidityPool implementation
 */

contract L2LiquidityPool is OVM_CrossDomainEnabled {
    /*************
     * Variables *
     *************/

    mapping(address => uint256) balances;
    mapping(address => uint256) fees;

    address owner;
    address L1LiquidityPoolAddress;
    uint256 fee;

    /********************************
     * Constructor & Initialization *
     ********************************/

    /**
     * @param _l2CrossDomainMessenger L1 Messenger address being used for cross-chain communications.
     */
    constructor(
        address _l2CrossDomainMessenger
    )
        OVM_CrossDomainEnabled(_l2CrossDomainMessenger)
    {
      owner = msg.sender;
    }

    /********************
     *       Event      *
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

    event clientDepositL2_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    event clientPayL2_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyOwner() {
        require(msg.sender == owner, "You don't own this contract");
        _;
    }

    modifier onlyInitialized() {
        require(address(L1LiquidityPoolAddress) != address(0), "Contract has not yet been initialized");
        _;
    }

    /********************
     * Public Functions *
     ********************/

    // Default gas value which can be overridden if more complex logic runs on L2.
    uint32 constant DEFAULT_FINALIZE_DEPOSIT_L2_GAS = 1200000;

    /**
     * @dev Initialize this contract with the L1 token gateway address.
     * The flow: 1) this contract gets deployed on L2, 2) the L1
     * gateway is deployed with addr from (1), 3) L1 gateway address passed here.
     *
     * @param _L1LiquidityPoolAddress Address of the corresponding L1 gateway deployed to the main chain
     * @param _fee Transaction fee
     */
    function init(
        address _L1LiquidityPoolAddress,
        uint256 _fee
    )
        public
        onlyOwner()
    {
        L1LiquidityPoolAddress = _L1LiquidityPoolAddress;
        fee = _fee;
    }

    /**
     * @dev Overridable getter for the *L2* gas limit of settling the deposit, in the case it may be
     * dynamic, and the above public constant does not suffice.
     *
     */

    function getFinalizeDepositL2Gas()
        public
        view
        virtual
        returns(
            uint32
        )
    {
        return DEFAULT_FINALIZE_DEPOSIT_L2_GAS;
    }

    /**
     * Checks the balance of an address.
     * @param _erc20ContractAddress Address of ERC20.
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
     * @param _amount Amount to transfer to the other account.
     * @param _erc20L2ContractAddress ERC20 L2 token address.
     */
    function ownerAddERC20Liquidity(
        uint256 _amount,
        address _erc20L2ContractAddress
    ) 
        external
        onlyOwner() 
    {
        ERC20 erc20Contract = ERC20(_erc20L2ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        require(erc20Contract.transferFrom(msg.sender, address(this), _amount), "ERC20 token transfer was unsuccessful");

        balances[_erc20L2ContractAddress] += _amount;

        emit ownerAddERC20Liquidity_EVENT(
            msg.sender,
            _amount,
            _erc20L2ContractAddress
        );
    }

    /**
     * Client deposit ERC20 from their account to this contract, which then releases funds on the L1 side
     * @param _amount Amount to transfer to the other account.
     * @param _erc20L2ContractAddress ERC20 token address
     */
    function clientDepositL2(
        uint256 _amount,
        address _erc20L2ContractAddress,
        address _erc20L1ContractAddress
    )
        external
    {
        ERC20 erc20Contract = ERC20(_erc20L2ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        require(erc20Contract.transferFrom(msg.sender, address(this), _amount), "ERC20 token transfer was unsuccessful");

        //Augment the pool size for this ERC20
        balances[_erc20L2ContractAddress] += _amount;

        // Construct calldata for L1LiquidityPool.depositToFinalize(_to, _amount)
        bytes memory data = abi.encodeWithSelector(
            L1LiquidityPool.clientPayL1.selector,
            msg.sender,
            _amount,
            _erc20L1ContractAddress
        );

        // Send calldata into L1
        sendCrossDomainMessage(
            address(L1LiquidityPoolAddress),
            data,
            getFinalizeDepositL2Gas()
        );

        emit clientDepositL2_EVENT(
            msg.sender,
            _amount,
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
        address _to
    )
        external
        onlyOwner()
    {
        ERC20 erc20Contract = ERC20(_erc20ContractAddress);
        require(fees[_erc20ContractAddress] >= _amount);
        require(erc20Contract.balanceOf(address(this)) >= _amount);
        require(erc20Contract.transfer(_to, _amount));
        fees[_erc20ContractAddress] -= _amount;

        emit ownerRecoverFee_EVENT(
            msg.sender,
            _to,
            _erc20ContractAddress,
            _amount
        );
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * Move funds from L1 to L2, and pay out from the right liquidity pool
     * @param _to Address to to be transferred.
     * @param _amount amount to to be transferred.
     * @param _erc20ContractAddress L2 erc20 token.
     */
    function clientPayL2(
        address _to,
        uint256 _amount,
        address _erc20ContractAddress
    )
        external
        onlyInitialized()
        onlyFromCrossDomainAccount(address(L1LiquidityPoolAddress))
    {
        ERC20 erc20Contract = ERC20(_erc20ContractAddress);
        uint256 _swapFee = _amount * fee / 100; //dangerous
        uint256 _receivedAmount = _amount - _swapFee; //need check
        require(erc20Contract.transfer(_to, _receivedAmount));

        balances[_erc20ContractAddress] -= _amount;
        fees[_erc20ContractAddress] += _swapFee;
        
        emit clientPayL2_EVENT(
          _to,
          _amount,
          _erc20ContractAddress
        );
    }
     
}
