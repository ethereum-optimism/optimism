// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

import { iL2LiquidityPool } from "./interfaces/iL2LiquidityPool.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "enyalabs_contracts/build/contracts/libraries/bridge/OVM_CrossDomainEnabled.sol";

/* External Imports */
import '@openzeppelin/contracts/math/SafeMath.sol';
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
/**
 * @dev An L1 LiquidityPool implementation
 */
contract L1LiquidityPool is OVM_CrossDomainEnabled {
    using SafeERC20 for IERC20;
    using SafeMath for uint256;
    uint256 constant internal SAFE_GAS_STIPEND = 2300;

    /*************
     * Variables *
     *************/

    // TO_DO
    // contract's balance for a token is unused, remove usage
    // can obtain balance of pool from token contract instead
    // modify to user balance map while allowing multiple lprovider support
    // mapping(address => uint256) balances;
    mapping(address => uint256) fees;
    // this is to stop attacks where caller specifies l2contractaddress
    // also acts as a whitelist
    mapping(address => address) l2ContractAddress;

    address owner;
    address l2LiquidityPoolAddress;
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
        l2ContractAddress[address(0)] = _l2ETHAddress;
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

    function registerTokenAddress(
        address _erc20L1ContractAddress,
        address _erc20L2ContractAddress
    )
        public
        onlyOwner()
    {
        // use with caution, can register only once
        require(l2ContractAddress[_erc20L1ContractAddress] == address(0), "Token Address Already Registerd");
        l2ContractAddress[_erc20L1ContractAddress] = _erc20L2ContractAddress;
    }

    /**
     * @dev Receive ETH
     *
     */
    receive() external payable {

        if (msg.sender != owner) {
            uint256 _swapFee = (msg.value.mul(fee)).div(100);
            uint256 _receivedAmount = msg.value.sub(_swapFee);

            fees[address(0)] = fees[address(0)].add(_swapFee);

            // Construct calldata for L2LiquidityPool.depositToFinalize(_to, _amount)
            bytes memory data = abi.encodeWithSelector(
                iL2LiquidityPool.clientPayL2.selector,
                msg.sender,
                _receivedAmount,
                l2ContractAddress[address(0)]
            );

            // Send calldata into L2
            sendCrossDomainMessage(
                l2LiquidityPoolAddress,
                data,
                getFinalizeDepositL2Gas()
            );
        }

        // balances[address(0)] = balances[address(0)].add(msg.value);
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

    // /**
    //  * Checks the balance of an address.
    //  * @param _erc20ContractAddress Address of ERC20
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
        require(l2ContractAddress[_erc20L1ContractAddress] != address(0), "Token L2 address not registered");
        IERC20 erc20Contract = IERC20(_erc20L1ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        erc20Contract.safeTransferFrom(msg.sender, address(this), _amount);

        // balances[_erc20L1ContractAddress] = balances[_erc20L1ContractAddress].add(_amount);

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
     */
    function clientDepositL1(
        uint256 _amount,
        address _erc20L1ContractAddress
    )
        external
    {
        require(l2ContractAddress[_erc20L1ContractAddress] != address(0), "Token L2 address not registered");
        IERC20 erc20Contract = IERC20(_erc20L1ContractAddress);
        require(_amount <= erc20Contract.allowance(msg.sender, address(this)));
        erc20Contract.safeTransferFrom(msg.sender, address(this), _amount);

        //Augment the pool size for this ERC20
        uint256 _swapFee = (_amount.mul(fee)).div(100);
        uint256 _receivedAmount = _amount.sub(_swapFee);
        // balances[_erc20L1ContractAddress] = balances[_erc20L1ContractAddress].add(_amount);
        fees[_erc20L1ContractAddress] = fees[_erc20L1ContractAddress].add(_swapFee);

        // Construct calldata for L2LiquidityPool.depositToFinalize(_to, _receivedAmount)
        bytes memory data = abi.encodeWithSelector(
            iL2LiquidityPool.clientPayL2.selector,
            msg.sender,
            _receivedAmount,
            l2ContractAddress[_erc20L1ContractAddress]
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
            l2ContractAddress[_erc20L1ContractAddress]
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
        if (_erc20ContractAddress != address(0)) {
            IERC20 erc20Contract = IERC20(_erc20ContractAddress);
            uint256 withdrawableLiquidity = (erc20Contract.balanceOf(address(this))).sub(fees[_erc20ContractAddress]);
            require(withdrawableLiquidity >= _amount, "Not enough liquidity on the pool to withdraw");
            // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
            // use safe erc20 transfer
            erc20Contract.safeTransfer(_to, _amount);
        } else {
            uint256 withdrawableLiquidity = (address(this).balance).sub(fees[_erc20ContractAddress]);
            require(withdrawableLiquidity >= _amount, "Not enough liquidity on the pool to withdraw");
            // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
            (bool sent,) = _to.call{gas: SAFE_GAS_STIPEND, value: _amount}("");
            require(sent, "Failed to send Ether");
        }

        emit ownerWithdrawLiquidity_EVENT(
            msg.sender, //which is == owner, otherwise would not have gotten here
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
        address payable _to
    )
        external
        onlyOwner()
    {
        if (_erc20ContractAddress != address(0)) {
            //we are dealing with an ERC20
            IERC20 erc20Contract = IERC20(_erc20ContractAddress);
            require(erc20Contract.balanceOf(address(this)) >= _amount);
            fees[_erc20ContractAddress] = fees[_erc20ContractAddress].sub(_amount);
            // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
            erc20Contract.safeTransfer(_to, _amount);
        } else {
            //we are dealing with Ether
            //address(this).balance is not supported
            // safety check on safemath
            // require(fees[address(0)] >= _amount);
            //_to.transfer(_amount); //unsafe
            // Call returns a boolean value indicating success or failure.
            // This is the current recommended method to use.
            //(bool sent,) = _to.call{value: msg.value}("");
            fees[address(0)] = fees[address(0)].sub(_amount);
            // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
            (bool sent,) = _to.call{gas: SAFE_GAS_STIPEND, value: _amount}("");
            require(sent, "Failed to send Ether");
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
            IERC20 erc20Contract = IERC20(_erc20ContractAddress);
            // balances[_erc20ContractAddress] = balances[_erc20ContractAddress].sub(_amount);
            erc20Contract.safeTransfer(_to, _amount);
        } else {
            //this is ETH
            // balances[address(0)] = balances[address(0)].sub(_amount);
            //_to.transfer(_amount); UNSAFE
            (bool sent,) = _to.call{gas: SAFE_GAS_STIPEND, value: _amount}("");
            require(sent, "Failed to send Ether");
        }

        emit clientPayL1_EVENT(
          _to,
          _amount,
          _erc20ContractAddress
        );
    }
}
