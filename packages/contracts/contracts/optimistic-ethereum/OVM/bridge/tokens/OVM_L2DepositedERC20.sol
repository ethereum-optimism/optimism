// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";
import { iOVM_L2DepositedToken } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedToken.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";

/* Contract Imports */
import { UniswapV2ERC20 } from "../../../libraries/standards/UniswapV2ERC20.sol";

/**
 * @title OVM_L2DepositedERC20
 * @dev The L2 Deposited ERC20 is an ERC20 implementation which represents L1 assets deposited into L2.
 * This contract mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
 * NOTE: This contract implements the Abs_L2DepositedToken contract using Uniswap's ERC20 as the implementation.
 * Alternative implementations can be used in this similar manner.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_L2DepositedERC20 is iOVM_L2DepositedToken, OVM_CrossDomainEnabled, UniswapV2ERC20 {
    /*******************
     * Contract Events *
     *******************/

    event Initialized(iOVM_L1TokenGateway _l1TokenGateway);

    /********************************
     * External Contract References *
     ********************************/

    iOVM_L1TokenGateway public l1TokenGateway;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l2CrossDomainMessenger Cross-domain messenger used by this contract.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    constructor(
        address _l2CrossDomainMessenger,
        string memory _name,
        string memory _symbol
    )
        OVM_CrossDomainEnabled(_l2CrossDomainMessenger)
        UniswapV2ERC20(_name, _symbol)
    {}

    /**
     * @dev Initialize this contract with the L1 token gateway address.
     *      The flow:
     *          1) this contract is deployed on L2,
     *          2) the L1 gateway is deployed with addr from (1),
     *          3) L1 gateway address passed here.
     * @param _l1TokenGateway Address of the corresponding L1 gateway deployed to the main chain
     */
    function init(
        iOVM_L1TokenGateway _l1TokenGateway
    )
        public
    {
        require(address(l1TokenGateway) == address(0), "Contract has already been initialized");

        l1TokenGateway = _l1TokenGateway;

        emit Initialized(l1TokenGateway);
    }

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyInitialized() {
        require(address(l1TokenGateway) != address(0), "Contract has not yet been initialized");
        _;
    }

    /**
     * @dev Core logic to be performed when a withdrawal from L2 is initialized.
     * When a withdrawal is initiated, we burn the withdrawer's funds to prevent subsequent L2 usage
     * param _to Address being withdrawn to.
     * param _amount Amount being withdrawn.
     */
    function _handleInitiateWithdrawal(
        address, // _to,
        uint256 _amount
    )
        internal
    {
        _burn(msg.sender, _amount);
    }

    /**
     * @dev Core logic to be performed when a deposit from L2 is finalized on L2.
     * When a deposit is finalized, we credit the account on L2 with the same amount of tokens.
     * param _to Address being deposited to on L2.
     * param _amount Amount which was deposited on L1.
     */
    function _handleFinalizeDeposit(
        address _to,
        uint256 _amount
    )
        internal
    {
        _mint(_to, _amount);
    }

    /***************
     * Withdrawing *
     ***************/

    /**
     * @dev initiate a withdraw of some tokens to the caller's account on L1
     * @param _amount Amount of the token to withdraw.
     * param _l1Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function withdraw(
        uint256 _amount,
        uint32, // _l1Gas,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateWithdrawal(
            msg.sender,
            msg.sender,
            _amount,
            0,
            _data
        );
    }

    /**
     * @dev initiate a withdraw of some token to a recipient's account on L1.
     * @param _to L1 adress to credit the withdrawal to.
     * @param _amount Amount of the token to withdraw.
     * param _l1Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function withdrawTo(
        address _to,
        uint256 _amount,
        uint32, // _l1Gas,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateWithdrawal(
            msg.sender,
            _to,
            _amount,
            0,
            _data
        );
    }

    /**
     * @dev Performs the logic for deposits by storing the token and informing the L2 token Gateway of the deposit.
     * @param _from Account to pull the deposit from on L2.
     * @param _to Account to give the withdrawal to on L1.
     * @param _amount Amount of the token to withdraw.
     * param _l1Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateWithdrawal(
        address _from,
        address _to,
        uint256 _amount,
        uint32, // _l1Gas,
        bytes calldata _data
    )
        internal
    {
        // Call our withdrawal accounting handler implemented by child contracts (usually a _burn)
        _handleInitiateWithdrawal(_to, _amount);

        // Construct calldata for l1TokenGateway.finalizeWithdrawal(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            iOVM_L1TokenGateway.finalizeWithdrawal.selector,
            _from,
            _to,
            _amount,
            _data
        );

        // Send message up to L1 gateway
        sendCrossDomainMessage(
            address(l1TokenGateway),
            0,
            message
        );

        emit WithdrawalInitiated(msg.sender, _to, _amount, _data);
    }

    /************************************
     * Cross-chain Function: Depositing *
     ************************************/

    /**
     * @dev Complete a deposit from L1 to L2, and credits funds to the recipient's balance of this
     * L2 token.
     * This call will fail if it did not originate from a corresponding deposit in OVM_l1TokenGateway.
     * @param _from Account to pull the deposit from on L2.
     * @param _to Address to receive the withdrawal at
     * @param _amount Amount of the token to withdraw
     * @param _data Data provider by the sender on L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function finalizeDeposit(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
        onlyFromCrossDomainAccount(address(l1TokenGateway))
    {
        _handleFinalizeDeposit(_to, _amount);
        emit DepositFinalized(_from, _to, _amount, _data);
    }
}
