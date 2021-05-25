// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

import { iL1LiquidityPool } from "./interfaces/iL1LiquidityPool.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "enyalabs_contracts/build/contracts/libraries/bridge/OVM_CrossDomainEnabled.sol";

/* External Imports */
import '@openzeppelin/contracts/math/SafeMath.sol';
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";

/**
 * @dev An L2 LiquidityPool implementation
 */

contract L2LiquidityPool is OVM_CrossDomainEnabled {
    using SafeERC20 for IERC20;
    using SafeMath for uint256;

    /*************
     * Variables *
     *************/

    // TO_DO
    // contract's balance for a token is unused, remove usage
    // can obtain balance of pool from token contract instead
    // modify to user balance map while allowing multiple lprovider support
    // mapping(address => uint256) balances;
    mapping(address => uint256) fees;
    // this is to stop attacks where caller specifies l1contractaddress
    // also acts as a whitelist
    mapping(address => address) l1ContractAddress;
    mapping(address => bool) isL2Eth;

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
        uint256 fee,
        address erc20ContractAddress
    );

    event clientPayL2_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    event ownerWithdrawLiquidity_EVENT(
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
        uint256 _fee,
        address l2ETHAddress
    )
        public
        onlyOwner()
    {
        L1LiquidityPoolAddress = _L1LiquidityPoolAddress;
        fee = _fee;
        isL2Eth[l2ETHAddress] = true;
    }

    function registerTokenAddress(
        address _erc20L1ContractAddress,
        address _erc20L2ContractAddress
    )
        public
        onlyOwner()
    {
        // use with caution, can register only once
        require(l1ContractAddress[_erc20L2ContractAddress] == address(0), "Token Address Already Registerd");
        require(!isL2Eth[_erc20L2ContractAddress], "Cannot replace Eth Address");
        l1ContractAddress[_erc20L2ContractAddress] = _erc20L1ContractAddress;
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

    // /**
    //  * Checks the balance of an address.
    //  * @param _erc20ContractAddress Address of ERC20.
    //  * @return Balance of the address.
    //  */
    // function balanceOf(
    //     address _erc20ContractAddress
    // )
    //     external
    //     view
    //     returns (
    //         uint256
    //     )
    // {
    //     return balances[_erc20ContractAddress];
    // }

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
        require(l1ContractAddress[_erc20L2ContractAddress] != address(0) || isL2Eth[_erc20L2ContractAddress], "Token Address Not Registerd");
        IERC20 erc20Contract = IERC20(_erc20L2ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        erc20Contract.safeTransferFrom(msg.sender, address(this), _amount);

        // balances[_erc20L2ContractAddress] = balances[_erc20L2ContractAddress].add(_amount);

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
        address _erc20L2ContractAddress
    )
        external
    {
        require(l1ContractAddress[_erc20L2ContractAddress] != address(0) || isL2Eth[_erc20L2ContractAddress], "Token Address Not Registerd");
        IERC20 erc20Contract = IERC20(_erc20L2ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        erc20Contract.safeTransferFrom(msg.sender, address(this), _amount);

        //Augment the pool size for this ERC20
        uint256 _swapFee = (_amount.mul(fee)).div(100);
        uint256 _receivedAmount = _amount.sub(_swapFee);
        // balances[_erc20L2ContractAddress] = balances[_erc20L2ContractAddress].add(_amount);
        fees[_erc20L2ContractAddress] = fees[_erc20L2ContractAddress].add(_swapFee);

        // Construct calldata for L1LiquidityPool.depositToFinalize(_to, _receivedAmount)
        bytes memory data = abi.encodeWithSelector(
            iL1LiquidityPool.clientPayL1.selector,
            msg.sender,
            _receivedAmount,
            l1ContractAddress[_erc20L2ContractAddress]
        );

        // Send calldata into L1
        sendCrossDomainMessage(
            address(L1LiquidityPoolAddress),
            data,
            getFinalizeDepositL2Gas()
        );

        emit clientDepositL2_EVENT(
            msg.sender,
            _receivedAmount,
            _amount,
            _erc20L2ContractAddress
        );

    }

    // only one liquidity provider
    // fees to be withdrawed from the other method, helps when shifting to multi Lproviders
    function ownerWithdrawLiqudiity(
        uint256 _amount,
        address _erc20ContractAddress,
        address payable _to
    )
        external
        onlyOwner()
    {
        IERC20 erc20Contract = IERC20(_erc20ContractAddress);
        uint256 withdrawableLiquidity = (erc20Contract.balanceOf(address(this))).sub(fees[_erc20ContractAddress]);
        require(withdrawableLiquidity >= _amount, "Not enough liquidity on the pool to withdraw");
        // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
        // use safe erc20 transfer
        erc20Contract.safeTransfer(_to, _amount);

        emit ownerWithdrawLiquidity_EVENT(
            msg.sender,
            _to,
            _erc20ContractAddress,
            _amount
        );
    }

    /**
     * owner recover fee from ERC20
     * @param _amount Amount to transfer to the other account.
     * @param _erc20ContractAddress ERC20 token address.
     * @param _to receiver to get the fee.
     */
    function ownerRecoverFee(
        uint256 _amount,
        address _erc20ContractAddress,
        address _to
    )
        external
        onlyOwner()
    {
        IERC20 erc20Contract = IERC20(_erc20ContractAddress);
        require(fees[_erc20ContractAddress] >= _amount);
        require(erc20Contract.balanceOf(address(this)) >= _amount);
        fees[_erc20ContractAddress] = fees[_erc20ContractAddress].sub(_amount);
        // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
        erc20Contract.safeTransfer(_to, _amount);

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
        IERC20 erc20Contract = IERC20(_erc20ContractAddress);
        // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
        erc20Contract.safeTransfer(_to, _amount);


        emit clientPayL2_EVENT(
          _to,
          _amount,
          _erc20ContractAddress
        );
    }

}
