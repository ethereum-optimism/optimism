// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721Bridge } from "../universal/ERC721Bridge.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { L2ERC721Bridge } from "../L2/L2ERC721Bridge.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @title L1ERC721Bridge
 * @notice The L1 ERC721 bridge is a contract which works together with the L2 ERC721 bridge to
 *         make it possible to transfer ERC721 tokens from Ethereum to Optimism. This contract
 *         acts as an escrow for ERC721 tokens deposted into L2.
 */
contract L1ERC721Bridge is ERC721Bridge, Semver {
    /**
     * @notice Mapping of L1 token to L2 token to ID to boolean, indicating if the given L1 token
     *         by ID was deposited for a given L2 token.
     */
    mapping(address => mapping(address => mapping(uint256 => bool))) public deposits;

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

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @notice Completes an ERC721 bridge from the other domain and sends the ERC721 token to the
     *         recipient on this domain.
     *
     * @param _localToken  Address of the ERC721 token on this domain.
     * @param _remoteToken Address of the ERC721 token on the other domain.
     * @param _from        Address that triggered the bridge on the other domain.
     * @param _to          Address to receive the token on this domain.
     * @param _tokenId     ID of the token being deposited.
     * @param _extraData   Optional data to forward to L2. Data supplied here will not be used to
     *                     execute any code on L2 and is only emitted as extra data for the
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
            // If the token ID for this L1/L2 NFT pair is not escrowed in the L1 Bridge or if
            // another error occurred during finalization, we initiate a cross-domain message to
            // send the NFT back to its original owner on L2. This can happen if an L2 native NFT is
            // bridged to L1, or if a user mistakenly entered an incorrect L1 ERC721 address.
            // In either case, we stop the process here and construct a withdrawal in which we
            // flip the to and from addresses. This ensures that event-based accounting
            // will indicate net-zero transfer to the recipient. The ERC721BridgeFailed event
            // emitted below can also be used to identify this occurence.
            bytes memory message = abi.encodeWithSelector(
                L2ERC721Bridge.finalizeBridgeERC721.selector,
                _remoteToken,
                _localToken,
                _to,
                _from, // Refund the NFT to the original owner on the remote chain.
                _tokenId,
                _extraData
            );

            // Send the message to the L2 bridge.
            // slither-disable-next-line reentrancy-events
            messenger.sendMessage(otherBridge, message, 600_000);

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
        require(_localToken != address(this), "L1ERC721Bridge: local token cannot be self");

        // Checks that the L1/L2 NFT pair has a token ID that is escrowed in the L1 Bridge.
        require(
            deposits[_localToken][_remoteToken][_tokenId] == true,
            "L1ERC721Bridge: Token ID is not escrowed in the L1 Bridge"
        );

        // Mark that the token ID for this L1/L2 token pair is no longer escrowed in the L1
        // Bridge.
        deposits[_localToken][_remoteToken][_tokenId] = false;

        // When a withdrawal is finalized on L1, the L1 Bridge transfers the NFT to the
        // withdrawer.
        IERC721(_localToken).safeTransferFrom(address(this), _to, _tokenId);
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

        // Construct calldata for _l2Token.finalizeBridgeERC721(_to, _tokenId)
        bytes memory message = abi.encodeWithSelector(
            L2ERC721Bridge.finalizeBridgeERC721.selector,
            _remoteToken,
            _localToken,
            _from,
            _to,
            _tokenId,
            _extraData
        );

        // Lock token into bridge
        deposits[_localToken][_remoteToken][_tokenId] = true;
        IERC721(_localToken).transferFrom(_from, address(this), _tokenId);

        // Send calldata into L2
        messenger.sendMessage(otherBridge, message, _minGasLimit);
        emit ERC721BridgeInitiated(_localToken, _remoteToken, _from, _to, _tokenId, _extraData);
    }
}
