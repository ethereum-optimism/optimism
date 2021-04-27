// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

import "./ERC20.sol";
import { L2LiquidityPool } from "./L2LiquidityPool.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "enyalabs_contracts/build/contracts/libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @dev A super simple LiquidityPool implementation!
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
     *       Event      *
     ********************/

    event initiateDepositedTo(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    event depositedTo(
        address sender,
        uint256 amount,
        address erc20ContractL1Address,
        address erc20ContractL2Address
    );

    event depositedToFinalized(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    event withdrewFee(
        address sender,
        address receiver,
        address erc20ContractAddress,
        uint256 amount
    );

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyOwner() {
        require(msg.sender == owner, "You don't own this contract");
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
            // Construct calldata for L2LiquidityPool.depositToFinalize(_to, _amount)
            bytes memory data = abi.encodeWithSelector(
                L2LiquidityPool.depositToFinalize.selector,
                msg.sender,
                msg.value,
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
     * add ERC20 balance to this smart contract!
     * @param _amount Amount to be transferred into this account.
     * @param _erc20L1ContractAddress ERC20 L1 token address.
     */
    function initiateDepositTo(
        uint256 _amount,
        address _erc20L1ContractAddress
    ) 
        external 
    {
        ERC20 erc20Contract = ERC20(_erc20L1ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        require(erc20Contract.transferFrom(msg.sender, address(this), _amount), "ERC20 token transfer was unsuccessful");

        balances[_erc20L1ContractAddress] += _amount;

        emit initiateDepositedTo(
            msg.sender,
            _amount,
            _erc20L1ContractAddress
        );
    }

    /**
     * deposit a balance from your account to this smart contract!
     * @param _amount Amount to transfer to the other account.
     * @param _erc20L1ContractAddress ERC20 L1 token address.
     * @param _erc20L2ContractAddress ERC20 L2 token address.
     */
    function depositTo(
        uint256 _amount,
        address _erc20L1ContractAddress,
        address _erc20L2ContractAddress
    )
        external
    {
        ERC20 erc20Contract = ERC20(_erc20L1ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        require(erc20Contract.transferFrom(msg.sender, address(this), _amount), "ERC20 token transfer was unsuccessful");

        balances[_erc20L1ContractAddress] += _amount;

        // Construct calldata for L2LiquidityPool.depositToFinalize(_to, _amount)
        bytes memory data = abi.encodeWithSelector(
            L2LiquidityPool.depositToFinalize.selector,
            msg.sender,
            _amount,
            _erc20L2ContractAddress
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2LiquidityPoolAddress,
            data,
            getFinalizeDepositL2Gas()
        );

        emit depositedTo(
            msg.sender,
            _amount,
            _erc20L1ContractAddress,
            _erc20L2ContractAddress
        );

    }

    /**
     * withdraw fee from ERC20
     * @param _amount Amount to transfer to the other account.
     * @param _erc20ContractAddress ERC20 token address.
     * @param _receiver receiver to get the fee.
     */
    function withdrawFee(
        uint _amount,
        address _erc20ContractAddress,
        address payable _receiver
    )
        external
        onlyOwner()
    {   
        if (_erc20ContractAddress != address(0)) {
            ERC20 erc20Contract = ERC20(_erc20ContractAddress);
            require(fees[_erc20ContractAddress] >= _amount);
            require(erc20Contract.balanceOf(address(this)) >= _amount);
            require(erc20Contract.transfer(_receiver, _amount));
        } else {
            // address(this).balance is not supported
            require(balances[_erc20ContractAddress] >= _amount);
            _receiver.transfer(_amount);
        }

        balances[_erc20ContractAddress] -= _amount;
        fees[_erc20ContractAddress] -= _amount;

        emit withdrewFee(
            msg.sender,
            _receiver,
            _erc20ContractAddress,
            _amount
        );
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * deposit a balance from this account to your account!
     * @param _receiver Address to to be transferred.
     * @param _amount amount to to be transferred.
     * @param _erc20ContractAddress L1 erc20 token.
     */
    function depositToFinalize(
        address payable _receiver,
        uint256 _amount,
        address _erc20ContractAddress
    )
        external
        onlyFromCrossDomainAccount(address(l2LiquidityPoolAddress))
    {   
        uint256 _swapFee = _amount * fee / 100;
        uint256 _receivedAmount = _amount - _swapFee;

        if (_erc20ContractAddress != address(0)) {
            ERC20 erc20Contract = ERC20(_erc20ContractAddress);
            require(erc20Contract.transfer(_receiver, _receivedAmount));

            balances[_erc20ContractAddress] -= _receivedAmount;
            fees[_erc20ContractAddress] += _swapFee;
        } else {
            _receiver.transfer(_receivedAmount);

            balances[_erc20ContractAddress] -= _receivedAmount;
            fees[_erc20ContractAddress] += _swapFee;   
        }
        
        emit depositedToFinalized(
          _receiver,
          _amount,
          _erc20ContractAddress
        );
    }
}
