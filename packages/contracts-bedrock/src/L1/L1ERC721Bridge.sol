// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721Bridge } from "src/universal/ERC721Bridge.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { L2ERC721Bridge } from "src/L2/L2ERC721Bridge.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { StandardBridge } from "src/universal/StandardBridge.sol";
import { Constants } from "src/libraries/Constants.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

/// @title L1ERC721Bridge
/// @notice The L1 ERC721 bridge is a contract which works together with the L2 ERC721 bridge to
///         make it possible to transfer ERC721 tokens from Ethereum to Optimism. This contract
///         acts as an escrow for ERC721 tokens deposited into L2.
contract L1ERC721Bridge is ERC721Bridge, ISemver {
    /// @notice Mapping of L1 token to L2 token to ID to boolean, indicating if the given L1 token
    ///         by ID was deposited for a given L2 token.
    mapping(address => mapping(address => mapping(uint256 => bool))) public deposits;

    /// @notice Address of the SuperchainConfig contract.
    SuperchainConfig public superchainConfig;

    /// @notice Semantic version.
    /// @custom:semver 2.1.0
    string public constant version = "2.1.0";

    /// @notice Constructs the L1ERC721Bridge contract.
    constructor() ERC721Bridge() {
        initialize({ _messenger: CrossDomainMessenger(address(0)), _superchainConfig: SuperchainConfig(address(0)) });
    }

    /// @notice Initializes the contract.
    /// @param _messenger   Contract of the CrossDomainMessenger on this network.
    /// @param _superchainConfig Contract of the SuperchainConfig contract on this network.
    function initialize(CrossDomainMessenger _messenger, SuperchainConfig _superchainConfig) public initializer {
        superchainConfig = _superchainConfig;
        __ERC721Bridge_init({
            _messenger: _messenger,
            _otherBridge: StandardBridge(payable(Predeploys.L2_ERC721_BRIDGE))
        });
    }

    /// @inheritdoc ERC721Bridge
    function paused() public view override returns (bool) {
        return superchainConfig.paused();
    }

    /// @notice Completes an ERC721 bridge from the other domain and sends the ERC721 token to the
    ///         recipient on this domain.
    /// @param _localToken  Address of the ERC721 token on this domain.
    /// @param _remoteToken Address of the ERC721 token on the other domain.
    /// @param _from        Address that triggered the bridge on the other domain.
    /// @param _to          Address to receive the token on this domain.
    /// @param _tokenId     ID of the token being deposited.
    /// @param _extraData   Optional data to forward to L2.
    ///                     Data supplied here will not be used to execute any code on L2 and is
    ///                     only emitted as extra data for the convenience of off-chain tooling.
    function finalizeBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _extraData
    )
        external
        onlyOtherBridge
    {
        require(paused() == false, "L1ERC721Bridge: paused");
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
        IERC721(_localToken).safeTransferFrom({ from: address(this), to: _to, tokenId: _tokenId });

        // slither-disable-next-line reentrancy-events
        emit ERC721BridgeFinalized(_localToken, _remoteToken, _from, _to, _tokenId, _extraData);
    }

    /// @inheritdoc ERC721Bridge
    function _initiateBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _tokenId,
        uint32 _minGasLimit,
        bytes calldata _extraData
    )
        internal
        override
    {
        require(_remoteToken != address(0), "L1ERC721Bridge: remote token cannot be address(0)");

        // Construct calldata for _l2Token.finalizeBridgeERC721(_to, _tokenId)
        bytes memory message = abi.encodeWithSelector(
            L2ERC721Bridge.finalizeBridgeERC721.selector, _remoteToken, _localToken, _from, _to, _tokenId, _extraData
        );

        // Lock token into bridge
        deposits[_localToken][_remoteToken][_tokenId] = true;
        IERC721(_localToken).transferFrom({ from: _from, to: address(this), tokenId: _tokenId });

        // Send calldata into L2
        messenger.sendMessage({ _target: address(otherBridge), _message: message, _minGasLimit: _minGasLimit });
        emit ERC721BridgeInitiated(_localToken, _remoteToken, _from, _to, _tokenId, _extraData);
    }
}
