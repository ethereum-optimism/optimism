// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721Bridge } from "../universal/ERC721Bridge.sol";
import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { L1ERC721Bridge } from "../L1/L1ERC721Bridge.sol";
import { IOptimismMintableERC721 } from "../universal/IOptimismMintableERC721.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @title L2ERC721Bridge
 * @notice The L2 ERC721 bridge is a contract which works together with the L1 ERC721 bridge to
 *         make it possible to transfer ERC721 tokens from Ethereum to Optimism. This contract
 *         acts as a minter for new tokens when it hears about deposits into the L1 ERC721 bridge.
 *         This contract also acts as a burner for tokens being withdrawn.
 *         **WARNING**: Do not bridge an ERC721 that was originally deployed on Optimism. This
 *         bridge ONLY supports ERC721s originally deployed on Ethereum. Users will need to
 *         wait for the one-week challenge period to elapse before their Optimism-native NFT
 *         can be refunded on L2.
 */
contract L2ERC721Bridge is ERC721Bridge, Semver {
    /**
     * @custom:semver 1.0.0
     *
     * @param _messenger   Address of the CrossDomainMessenger on this network.
     * @param _otherBridge Address of the ERC721 bridge on the other network.
     */
    constructor(address _messenger, address _otherBridge)
        Semver(1, 0, 0)
        ERC721Bridge(_messenger, _otherBridge)
    {}

    /**
     * @notice Completes an ERC721 bridge from the other domain and sends the ERC721 token to the
     *         recipient on this domain.
     *
     * @param _localToken  Address of the ERC721 token on this domain.
     * @param _remoteToken Address of the ERC721 token on the other domain.
     * @param _from        Address that triggered the bridge on the other domain.
     * @param _to          Address to receive the token on this domain.
     * @param _tokenId     ID of the token being deposited.
     * @param _extraData   Optional data to forward to L1. Data supplied here will not be used to
     *                     execute any code on L1 and is only emitted as extra data for the
     *                     convenience of off-chain tooling.
     */
    function finalizeBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _extraData
    ) external onlyOtherBridge {
        try this.completeOutboundTransfer(_localToken, _remoteToken, _to, _tokenId) {
            // slither-disable-next-line reentrancy-events
            emit ERC721BridgeFinalized(_localToken, _remoteToken, _from, _to, _tokenId, _extraData);
        } catch {
            // Either the L2 token which is being deposited-into disagrees about the correct address
            // of its L1 token, or does not support the correct interface.
            // This should only happen if there is a malicious L2 token, or if a user somehow
            // specified the wrong L2 token address to deposit into.
            // There is no way to prevent malicious token contracts altogether, but this does limit
            // user error and mitigate some forms of malicious contract behavior.
            /// In either case, we stop the process here and construct a withdrawal in which we
            // flip the to and from addresses. This ensures that event-based accounting
            // will indicate net-zero transfer to the recipient. The ERC721BridgeFailed event
            // emitted below can also be used to identify this occurence.
            bytes memory message = abi.encodeWithSelector(
                L1ERC721Bridge.finalizeBridgeERC721.selector,
                _remoteToken,
                _localToken,
                _to,
                _from, // Refund the NFT to the original owner on the remote chain.
                _tokenId,
                _extraData
            );

            // Send the message to the L1 bridge
            // slither-disable-next-line reentrancy-events
            messenger.sendMessage(otherBridge, message, 0);

            // slither-disable-next-line reentrancy-events
            emit ERC721BridgeFailed(_localToken, _remoteToken, _from, _to, _tokenId, _extraData);
        }
    }

    /**
     * @notice Completes an outbound token transfer. Public function, but can only be called by
     *         this contract. It's security critical that there be absolutely no way for anyone to
     *         trigger this function, except by explicit trigger within this contract. Used as a
     *         simple way to be able to try/catch any type of revert that could occur during an
     *         ERC721 mint/transfer.
     *
     * @param _localToken  Address of the ERC721 on this chain.
     * @param _remoteToken Address of the corresponding token on the remote chain.
     * @param _to          Address of the receiver.
     * @param _tokenId     ID of the token being deposited.
     */
    function completeOutboundTransfer(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _tokenId
    ) external onlySelf {
        require(_localToken != address(this), "L2ERC721Bridge: local token cannot be self");

        require(
            // Note that supportsInterface makes a callback to the _localToken address
            // which is user provided.
            ERC165Checker.supportsInterface(_localToken, type(IOptimismMintableERC721).interfaceId),
            "L2ERC721Bridge: local token interface is not compliant"
        );
        require(
            _remoteToken == IOptimismMintableERC721(_localToken).remoteToken(),
            "L2ERC721Bridge: wrong remote token for Optimism Mintable ERC721 local token"
        );

        // When a deposit is finalized, we give the NFT with the same tokenId to the account
        // on L2. Note that safeMint makes a callback to the _to address which is user provided.
        IOptimismMintableERC721(_localToken).safeMint(_to, _tokenId);
    }

    /**
     * @inheritdoc ERC721Bridge
     */
    function _initiateBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _tokenId,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) internal override {
        require(_remoteToken != address(0), "ERC721Bridge: remote token cannot be address(0)");

        // Check that the withdrawal is being initiated by the NFT owner
        require(
            _from == IOptimismMintableERC721(_localToken).ownerOf(_tokenId),
            "Withdrawal is not being initiated by NFT owner"
        );

        // Construct calldata for l1ERC721Bridge.finalizeBridgeERC721(_to, _tokenId)
        // slither-disable-next-line reentrancy-events
        address remoteToken = IOptimismMintableERC721(_localToken).remoteToken();
        require(
            remoteToken == _remoteToken,
            "L2ERC721Bridge: remote token does not match given value"
        );

        // When a withdrawal is initiated, we burn the withdrawer's NFT to prevent subsequent L2
        // usage
        // slither-disable-next-line reentrancy-events
        IOptimismMintableERC721(_localToken).burn(_from, _tokenId);

        bytes memory message = abi.encodeWithSelector(
            L1ERC721Bridge.finalizeBridgeERC721.selector,
            remoteToken,
            _localToken,
            _from,
            _to,
            _tokenId,
            _extraData
        );

        // Send message to L1 bridge
        // slither-disable-next-line reentrancy-events
        messenger.sendMessage(otherBridge, message, _minGasLimit);

        // slither-disable-next-line reentrancy-events
        emit ERC721BridgeInitiated(_localToken, remoteToken, _from, _to, _tokenId, _extraData);
    }
}
