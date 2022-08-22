// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721Bridge } from "../universal/op-erc721/ERC721Bridge.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { L2ERC721Bridge } from "../L2/L2ERC721Bridge.sol";
import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

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
     * @custom:semver 0.0.1
     *
     * @param _messenger   Address of the CrossDomainMessenger on this network.
     * @param _otherBridge Address of the ERC721 bridge on the other network.
     */
    constructor(address _messenger, address _otherBridge) Semver(0, 0, 1) {
        initialize(_messenger, _otherBridge);
    }

    /**
     * @notice Initializer.
     *
     * @param _messenger   Address of the L1CrossDomainMessenger.
     * @param _otherBridge Address of the L2ERC721Bridge.
     */
    function initialize(address _messenger, address _otherBridge) public initializer {
        require(_messenger != address(0), "ERC721Bridge: messenger cannot be address(0)");
        require(_otherBridge != address(0), "ERC721Bridge: other bridge cannot be address(0)");
        __ERC721Bridge_init(_messenger, _otherBridge);
    }

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
            if (_from == otherBridge) {
                // The _from address is the address of the remote bridge if a transfer fails to be
                // finalized on the remote chain.
                // slither-disable-next-line reentrancy-events
                emit ERC721Refunded(_localToken, _remoteToken, _to, _tokenId, _extraData);
            } else {
                // slither-disable-next-line reentrancy-events
                emit ERC721BridgeFinalized(
                    _localToken,
                    _remoteToken,
                    _from,
                    _to,
                    _tokenId,
                    _extraData
                );
            }
        } catch {
            // If the token ID for this L1/L2 NFT pair is not escrowed in the L1 Bridge or if
            // another error occurred during finalization, we initiate a cross-domain message to
            // send the NFT back to its original owner on L2. This can happen if an L2 native NFT is
            // bridged to L1, or if a user mistakenly entered an incorrect L1 ERC721 address.
            bytes memory message = abi.encodeWithSelector(
                L2ERC721Bridge.finalizeBridgeERC721.selector,
                _remoteToken,
                _localToken,
                address(this), // Set the new _from address to be this contract since the NFT was
                // never transferred to the recipient on this chain.
                _from, // Refund the NFT to the original owner on the remote chain.
                _tokenId,
                _extraData
            );

            // Send the message to the L2 bridge.
            // slither-disable-next-line reentrancy-events
            sendCrossDomainMessage(otherBridge, 600_000, message);

            // slither-disable-next-line reentrancy-events
            emit ERC721BridgeFailed(_localToken, _remoteToken, _from, _to, _tokenId, _extraData);
        }
    }

    /**
     * @inheritdoc ERC721Bridge
     */
    function completeOutboundTransfer(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _tokenId
    ) external onlySelf {
        // Checks that the L1/L2 NFT pair has a token ID that is escrowed in the L1 Bridge. Without
        // this check, an attacker could steal a legitimate L1 NFT by supplying an arbitrary L2 NFT
        // that maps to the L1 NFT.
        require(
            deposits[_localToken][_remoteToken][_tokenId] == true,
            "L1ERC721Bridge: token ID is not escrowed in l1 bridge for this l1/l2 nft pair"
        );

        // Mark that the token ID for this L1/L2 token pair is no longer escrowed in the L1
        // Bridge.
        deposits[_localToken][_remoteToken][_tokenId] = false;

        // When a withdrawal is finalized on L1, the L1 Bridge transfers the NFT to the
        // withdrawer.
        IERC721(_localToken).safeTransferFrom(address(this), _to, _tokenId);
    }

    /**
     * @notice Internal function for initiating a token bridge to the other domain.
     *
     * @param _localToken  Address of the ERC721 on this domain.
     * @param _remoteToken Address of the ERC721 on the remote domain.
     * @param _from        Address of the sender on this domain.
     * @param _to          Address to receive the token on the other domain.
     * @param _tokenId     Token ID to bridge.
     * @param _minGasLimit Minimum gas limit for the bridge message on the other domain.
     * @param _extraData   Optional data to forward to L2. Data supplied here will not be used to
     *                     execute any code on L2 and is only emitted as extra data for the
     *                     convenience of off-chain tooling.
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
