// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L1NFTBridge
 */
interface iOVM_L1NFTBridge {

    event NFTDepositInitiated (
        address indexed _l1Contract,
        address indexed _l2Contract,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        bytes _data
    );

    event NFTWithdrawalFinalized (
        address indexed _l1Contract,
        address indexed _l2Contract,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        bytes _data
    );

    function depositNFT(
        address _l1Contract,
        address _l2Contract,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external;

    function depositNFTTo(
        address _l1Contract,
        address _l2Contract,
        address _to,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external;

    function finalizeNFTWithdrawal(
        address _l1Contract,
        address _l2Contract,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _data
    )
        external;

}
