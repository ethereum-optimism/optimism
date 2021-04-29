// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_TokenGateway.sol";

/* Contract Imports */
import { OVM_L2ERC20 } from "../../../libraries/standards/OVM_L2ERC20.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";


/**
 * @title OVM_L2TokenGateway
 * @dev The L2 Token Gateway is an ERC20 implementation which represents L1 assets deposited into L2.
 * This contract mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
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
    uint32 internal constant DEFAULT_L1_FINALIZATION_GAS = 100000;


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
        OVM_L2ERC20 _l2ERC20,
        string memory _name,
        string memory _symbol
    )
        OVM_CrossDomainEnabled(_l2CrossDomainMessenger)
    {
        if(address(_l2ERC20) == address(0) ){
            l2ERC20 = new OVM_L2ERC20(_name, _symbol);
        } else {
            l2ERC20 = _l2ERC20;
        }
    }

    /**
     * @dev Initialize this contract with the L1 token gateway address.
     * The flow:
     * 1. this contract gets deployed on L2,
     * 2. the L1 gateway is deployed with the address from step 1,
     * 3. the L1 gateway address passed here.
     *
     * @param _l1TokenGateway Address of the corresponding L1 gateway deployed to the main chain
     */
    function init(
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


    /********************
     * Getter Functions *
     *******************/

    /**
     * @dev Get the address of the gateway on the *cross-domain*.
     * @return Address.
     */
    function crossDomainGateway()
        external
        view
        override
        virtual
        returns(address)
    {
        return address(l1TokenGateway);
    }

    /**
     * @dev Overridable getter for the *cross-domain* gas limit of settling the outbound transfer,
     * in the case it may be dynamic, and the above public constant does not suffice.
     * @return L1 finalization gas required.
     */
    function getL1GasToFinalize()
        public
        view
        virtual
        returns(
            uint32
        )
    {
        return DEFAULT_L1_FINALIZATION_GAS;
    }

    /**********************
     * External Functions *
     *********************/

    /**
     * @dev initiate an outboundTransfer of the token to the caller's account on L1
     * @param _amount Amount of the token to outboundTransfer
     */
    function outboundTransfer(
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateOutboundTransfer(msg.sender, msg.sender, _amount, _data);
    }

    /**
     * @dev initiate an outboundTransfer of the token to a recipient's account on L1
     * @param _to L1 adress to credit the outboundTransfer to.
     * @param _amount Amount of the token to outboundTransfer.
     */
    function outboundTransferTo(
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateOutboundTransfer(msg.sender, _to, _amount, _data);
    }

    /**
     * @dev Complete a deposit from L1 to L2, and credits funds to the recipient's balance of this
     * L2 token.
     * This call will fail if it did not originate from a corresponding deposit in OVM_l1TokenGateway.
     *
     * @param _to Address to receive the withdrawal at.
     * @param _amount Amount of the token to withdraw.
     * @return Address of the sender on L1.
     * @return Data provided with the message from L1.
     */
    function finalizeInboundTransfer(
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

        emit InboundTransfer(_from, _to, _amount, _data);

        // @todo: I don't actually see any good reason to return this data.
        // the caller already has it, but it's in the interface proposal, so keeping it for now.
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
    function _initiateOutboundTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        internal
    {
        // Call our withdrawal accounting handler implemented by child contracts (usually a _burn)
        l2ERC20.burn(_from, _amount);

        // Construct calldata for l1TokenGateway.finalizeInboundTransfer(_to, _amount)
        bytes memory data = abi.encodeWithSelector(
            iOVM_TokenGateway.finalizeInboundTransfer.selector,
            _to,
            _amount
        );

        // Send message up to L1 gateway
        sendCrossDomainMessage(
            address(l1TokenGateway),
            data,
            getL1GasToFinalize()
        );

        emit OutboundTransfer(msg.sender, _to, _amount, _data);
    }
}
