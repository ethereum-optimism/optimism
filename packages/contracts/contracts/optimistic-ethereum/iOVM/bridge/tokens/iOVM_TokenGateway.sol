// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_TokenGateway
 */
interface iOVM_TokenGateway {

    /**********
     * Events *
     **********/

	/**
	 * @dev This emits when a token is sent from this domain to the cross-domain,
	 * ie. in transferOver() and transferOverTo().
	 * @param _from Address tokens were sent from.
	 * @param _to Address that will receive the tokens on the cross-domain.
    * @param _amount Amount of the token to transfer.
    */
    event TransferredOver(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _data
    );

	/**
	 * @dev FinalizedReturn is emitted when a token transfer from the cross-domain is paid out on this domain.
     * ie. in finalizeReturnTransfer().
	 * @param _from Address tokens were sent from on the cross-domain.
	 * @param _to Address that will receive the tokens on the cross-domain.
     * @param _amount Amount of the token to transfer.
     */
    event FinalizedReturn(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _data
    );


    /********************
     * Public Functions *
     ********************/


    /**
     * @return Address of the token (or the gateway)? on the xDomain
     */
    function crossDomainToken() // @todo: Token or Gateway?
        external
        returns
    (
        address
    );

	/**
	 * @notice Transfers a token to the same address as msg.sender on the cross-domain.
	 * emits a TransferredOver event
     * @param _amount Amount of the ERC20 to deposit.
     * @param _data Arbitrary data with additional information for use on the cross-domain.
     */
    function transferOver(
        uint _amount,
        bytes calldata _data
    )
        external;

	/**
	 * @notice Transfers a token to another address on the cross-domain
	 * emits a TransferredOver event
	 * @param _to Address on cross domain to transfer to.
     * @param _amount Amount of the ERC20 to transfer.
     * @param _data Arbitrary data with additional information for use on the cross-domain.
     */
    function transferOverTo(
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;


    /************************
    * Cross-chain Functions *
    ************************/

	/**
	 * @notice Finalizes one or more transfers initiated on the cross-domain
	 * emits a FinalizedReturn event.
	 * @param _to Address to transfer the token to.
     * @param _amount Amount of the ERC20 to transfer.
     * @return _from Address of the sender on the cross-domain.
     * @return _data Data with additional information for use on the cross-domain.
     */
    function finalizeReturnTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external
        returns
    (
        address,
        bytes memory
    );
}

