// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L1ETHGateway
 */
interface iOVM_L1ETHGateway {

    /**********
     * Events *
     **********/

    event OutboundTransferInitiated(
        address indexed _from,
        address _to,
        uint256 _amount
    );

    event InboundTransferFinalized(
        address indexed _to,
        uint256 _amount
    );

    /********************
     * Public Functions *
     ********************/

    function outboundTransfer(
        bytes calldata _data
    )
        external
        payable;

    function outboundTransferTo(
        address _to,
        bytes calldata _data
    )
        external
        payable;

    /*************************
     * Cross-chain Functions *
     *************************/

    function finalizeInboundTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data

    )
        external;

    function getFinalizationGas()
        external
        view
        returns(
            uint32
        );
}
