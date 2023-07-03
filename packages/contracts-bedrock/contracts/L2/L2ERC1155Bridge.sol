// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { ERC1155Bridge } from "../universal/ERC1155Bridge.sol";
import { L1ERC1155Bridge } from "../L1/L1ERC1155Bridge.sol";
import { IOptimismMintableERC1155 } from "../universal/IOptimismMintableERC1155.sol";
import { Semver } from "../universal/Semver.sol";

/// @title L2ERC1155Bridge
/// @notice The L2 ERC1155 bridge is a contract which works together with the L1 ERC1155 bridge to
///         make it possible to transfer ERC1155 tokens from Ethereum to Optimism. This contract
///         acts as a minter for new tokens when it hears about deposits into the L1 ERC1155 bridge.
///         This contract also acts as a burner for tokens being withdrawn.
///         **WARNING**: Do not bridge an ERC1155 that was originally deployed on Optimism. This
///         bridge ONLY supports ERC1155s originally deployed on Ethereum. Users will need to
///         wait for the one-week challenge period to elapse before their Optimism-native NFT
///         can be refunded on L2.
contract L2ERC1155Bridge is ERC1155Bridge, Semver {
    /// @custom:semver 1.0.0
    /// @notice Constructs the L2ERC1155Bridge contract.
    /// @param _messenger   Address of the CrossDomainMessenger on this network.
    /// @param _otherBridge Address of the ERC1155 bridge on the other network.
    constructor(address _messenger, address _otherBridge)
        Semver(1, 0, 0)
        ERC1155Bridge(_messenger, _otherBridge)
    {}

    /// @notice Completes an ERC1155 bridge from the other domain and sends the ERC1155 token to the
    ///         recipient on this domain.
    /// @param _localToken  Address of the ERC1155 token on this domain.
    /// @param _remoteToken Address of the ERC1155 token on the other domain.
    /// @param _from        Address that triggered the bridge on the other domain.
    /// @param _to          Address to receive the token on this domain.
    /// @param _id          Type ID of the token being deposited.
    /// @param _amount      Amount of tokens to bridge.
    /// @param _extraData   Optional data to forward to L1.
    ///                     Data supplied here will not be used to execute any code on L1 and is
    ///                     only emitted as extra data for the convenience of off-chain tooling.
    function finalizeBridgeERC1155(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _id,
        uint256 _amount,
        bytes calldata _extraData
    ) external onlyOtherBridge {
        require(_localToken != address(this), "L2ERC1155Bridge: local token cannot be self");

        // Note that supportsInterface makes a callback to the _localToken address which is user
        // provided.
        require(
            ERC165Checker.supportsInterface(
                _localToken,
                type(IOptimismMintableERC1155).interfaceId
            ),
            "L2ERC1155Bridge: local token interface is not compliant"
        );

        require(
            _remoteToken == IOptimismMintableERC1155(_localToken).remoteToken(),
            "L2ERC1155Bridge: wrong remote token for Optimism Mintable ERC1155 local token"
        );

        // When a deposit is finalized, we give the amount of the same type ID to the account
        // on L2. Note that mint makes a callback to the _to address which is user provided.
        IOptimismMintableERC1155(_localToken).mint(_to, _id, _amount);

        emit ERC1155BridgeFinalized(
            _localToken,
            _remoteToken,
            _from,
            _to,
            _id,
            _amount,
            _extraData
        );
    }

    /// @inheritdoc ERC1155Bridge
    function _initiateBridgeERC1155(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _id,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) internal override {
        require(_remoteToken != address(0), "L2ERC1155Bridge: remote token cannot be address(0)");

        // Construct calldata for l1ERC1155Bridge.finalizeBridgeERC1155(_to, _id, _amount)
        // slither-disable-next-line reentrancy-events
        address remoteToken = IOptimismMintableERC1155(_localToken).remoteToken();
        require(
            remoteToken == _remoteToken,
            "L2ERC1155Bridge: remote token does not match given value"
        );

        // When a withdrawal is initiated, we burn the amount of the type ID to prevent subsequent
        // L2 usage
        // slither-disable-next-line reentrancy-events
        IOptimismMintableERC1155(_localToken).burn(_from, _id, _amount);

        bytes memory message = abi.encodeWithSelector(
            L1ERC1155Bridge.finalizeBridgeERC1155.selector,
            remoteToken,
            _localToken,
            _from,
            _to,
            _id,
            _amount,
            _extraData
        );

        // Send message to L1 bridge
        // slither-disable-next-line reentrancy-events
        MESSENGER.sendMessage(OTHER_BRIDGE, message, _minGasLimit);

        // slither-disable-next-line reentrancy-events
        emit ERC1155BridgeInitiated(_localToken, remoteToken, _from, _to, _id, _amount, _extraData);
    }
}
