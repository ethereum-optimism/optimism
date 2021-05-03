// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";

/* Contract Imports */
import { OVM_L2ERC20 } from "../../../libraries/standards/OVM_L2ERC20.sol";

/* Library Imports */
import { Abs_L2TokenGateway } from "./Abs_L2TokenGateway.sol";

/**
 * @title OVM_L2TokenGateway
 * @dev The Token Gateway facilitates depositing L1 assets into L2.
 * This contract controls an L2 ERC20 token, and mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_L2TokenGateway is Abs_L2TokenGateway {

    /*********************
     * Storage Variables *
     ********************/
    OVM_L2ERC20 public token;

    /***************
     * Constructor *
     ***************/

    /**
     * @dev A token address may either be provided, or a new one will be created.
     * @param _l2CrossDomainMessenger Cross-domain messenger used by this contract.
     * @param _token ERC20 token address
     * @param _name ERC20 name
     * @param _symbol ERC20 symbol
     */
    constructor(
        address _l2CrossDomainMessenger,
        address _token,
        string memory _name,
        string memory _symbol
    )
        Abs_L2TokenGateway(_l2CrossDomainMessenger)
    {
        if(_token == address(0)){
            token = new OVM_L2ERC20(
                _name,
                _symbol
            );
        } else {
            token = OVM_L2ERC20(_token);
        }
    }

    // When a withdrawal is initiated, we burn the withdrawer's funds to prevent subsequent L2 usage.
    function _handleInitiateOutboundTransfer(
        address, // _to,
        uint _amount
    )
        internal
        override
    {
        token.burn(msg.sender, _amount);
    }

    // When a deposit is finalized, we credit the account on L2 with the same amount of tokens.
    function _handleFinalizeInboundTransfer(
        address _to,
        uint _amount
    )
        internal
        override
    {
        token.mint(_to, _amount);
    }
}
