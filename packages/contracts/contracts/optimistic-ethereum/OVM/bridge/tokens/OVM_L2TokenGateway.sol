// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_TokenGateway.sol";

/* Contract Imports */
// TODO: get a generic mint/burn token
import { OVM_L2ERC20 } from "../../../libraries/standards/OVM_L2ERC20.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";


/**
 * @title OVM_L2TokenGateway
 * @dev The L2 Token Gateway is an ERC20 implementation which represents L1 assets deposited into L2.
 * This contract mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
 * NOTE: This contract implements the Abs_L2DepositedToken contract using Uniswap's ERC20 as the implementation.
 * Alternative implementations can be used in this similar manner.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_L2TokenGateway is iOVM_TokenGateway, OVM_CrossDomainEnabled  {

    /*******************
     * Contract Events *
     *******************/

    event Initialized(iOVM_TokenGateway _l1TokenGateway);

    /*************
     * Constants *
     *************/

    // Default gas value which can be overridden if more complex logic runs on the cross-domain.
    uint32 internal constant DEFAULT_FINALIZE_TRANSFER_OVER_L1_GAS = 100000;


    /********************************
     * External Contract References *
     ********************************/

    iOVM_TokenGateway public l1TokenGateway;
    OVM_L2ERC20 public l2ERC20; // @todo: I'd prefer this be an interface

    /********************************
     * Constructor & Initialization *
     ********************************/

    /**
     * @param _l2CrossDomainMessenger Cross-domain messenger used by this contract.
     * @param _name ERC20 name
     * @param _symbol ERC20 symbol
     */
    constructor(
        address _l2CrossDomainMessenger,
        string memory _name,
        string memory _symbol
    )
        OVM_CrossDomainEnabled(_l2CrossDomainMessenger)
    {}

    /**
     * @dev Initialize this contract with the L1 token gateway address.
     * The flow: 1) this contract gets deployed on L2, 2) the L1
     * gateway is deployed with addr from (1), 3) L1 gateway address passed here.
     *
     * @param _l1TokenGateway Address of the corresponding L1 gateway deployed to the main chain
     */
    function initialize(
        iOVM_TokenGateway _l1TokenGateway
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


    /*****************************
     * External Getter Functions *
     *****************************/

    /**
     * @dev Get the address of the gateway/(@todo: or token???) on the *cross-domain*.
     */
    function crossDomainToken()
        external
        view
        override
        virtual
        returns(address)
    {
        return address(l2ERC20);
    }

    /**
     * @dev Overridable getter for the *cross-domain* gas limit of settling the transfer over, in the case it may be
     * dynamic, and the above public constant does not suffice.
     */
    function getL1GasToFinalize()
        public
        view
        virtual
        returns(
            uint32
        )
    {
        return DEFAULT_FINALIZE_TRANSFER_OVER_L1_GAS;
    }

    /**
     * @dev initiate a "transferOver" of the token to the caller's account on L1
     * @param _amount Amount of the token to transferOver
     */
    function transferOver(
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateTransferOver(msg.sender, msg.sender, _amount, _data);
    }

    /**
     * @dev initiate a transferOver of the token to a recipient's account on L1
     * @param _to L1 adress to credit the transferOver to
     * @param _amount Amount of the token to transferOver
     */
    function transferOverTo(
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateTransferOver(msg.sender, _to, _amount, _data);
    }

    /**
     * @dev Complete a deposit from L1 to L2, and credits funds to the recipient's balance of this
     * L2 token.
     * This call will fail if it did not originate from a corresponding deposit in OVM_l1TokenGateway.
     *
     * @param _to Address to receive the withdrawal at
     * @param _amount Amount of the token to withdraw
     */
    function finalizeReturnTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
        onlyFromCrossDomainAccount(address(l1TokenGateway))
        returns
        (
            address,
            bytes memory
        )
    {
        l2ERC20.mint(_to, _amount);

        emit FinalizedReturn(_from, _to, _amount, _data);

        return (_from, _data);
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * @dev Performs the logic for deposits by storing the token and informing the L2 token Gateway of the deposit.
     *
     * @param _to Account to give the withdrawal to on L1
     * @param _amount Amount of the token to withdraw
     */
    function _initiateTransferOver(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        internal
    {
        // Call our withdrawal accounting handler implemented by child contracts (usually a _burn)
        l2ERC20.burn(_from, _amount);

        // Construct calldata for l1TokenGateway.finalizeReturnTransfer(_to, _amount)
        bytes memory data = abi.encodeWithSelector(
            iOVM_TokenGateway.finalizeReturnTransfer.selector,
            _to,
            _amount
        );

        // Send message up to L1 gateway
        sendCrossDomainMessage(
            address(l1TokenGateway),
            data,
            getL1GasToFinalize()
        );

        emit TransferredOver(msg.sender, _to, _amount, _data);
    }
}
