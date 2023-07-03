// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IERC1155 } from "@openzeppelin/contracts/token/ERC1155/IERC1155.sol";
import { ERC1155Holder } from "@openzeppelin/contracts/token/ERC1155/utils/ERC1155Holder.sol";
import { L2ERC1155Bridge } from "../L2/L2ERC1155Bridge.sol";
import { ERC1155Bridge } from "../universal/ERC1155Bridge.sol";
import { Semver } from "../universal/Semver.sol";

/// @title L1ERC1155Bridge
/// @notice The L1 ERC1155 bridge is a contract which works together with the L2 ERC1155 bridge to
///         make it possible to transfer ERC1155 tokens from Ethereum to Optimism. This contract
///         acts as an escrow for ERC1155 tokens deposited into L2.
contract L1ERC1155Bridge is ERC1155Bridge, ERC1155Holder, Semver {
    /// @notice Mapping of L1 token to L2 token to type ID to deposits, indicating the amount of
    ///         L1 tokens by type ID deposited for L2 tokens.
    mapping(address => mapping(address => mapping(uint256 => uint256))) public deposits;

    /// @custom:semver 1.0.0
    /// @notice Constructs the L1ERC1155Bridge contract.
    /// @param _messenger   Address of the CrossDomainMessenger on this network.
    /// @param _otherBridge Address of the ERC1155 bridge on the other network.
    constructor(address _messenger, address _otherBridge)
        Semver(1, 0, 0)
        ERC1155Bridge(_messenger, _otherBridge)
    {}

    /// @notice Completes an ERC1155 bridge from the other domain and sends the ERC1155 tokens to
    ///         the recipient on this domain.
    /// @param _localToken  Address of the ERC1155 token on this domain.
    /// @param _remoteToken Address of the ERC1155 token on the other domain.
    /// @param _from        Address that triggered the bridge on the other domain.
    /// @param _to          Address to receive the token on this domain.
    /// @param _id          Type ID of the token being deposited.
    /// @param _amount      Amount of tokens to bridge.
    /// @param _extraData   Optional data to forward to L2.
    ///                     Data supplied here will not be used to execute any code on L2 and is
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
        require(_localToken != address(this), "L1ERC1155Bridge: local token cannot be self");

        // Reduce amount locked for token type ID for this L1/L2 token pair
        deposits[_localToken][_remoteToken][_id] -= _amount;

        // When a withdrawal is finalized on L1, the L1 Bridge transfers the NFT to the
        // withdrawer.
        IERC1155(_localToken).safeTransferFrom(address(this), _to, _id, _amount, "");

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
        require(_remoteToken != address(0), "L1ERC1155Bridge: remote token cannot be address(0)");

        // Construct calldata for _l2Bridge.finalizeBridgeERC1155(_to, _id, _amount)
        bytes memory message = abi.encodeWithSelector(
            L2ERC1155Bridge.finalizeBridgeERC1155.selector,
            _remoteToken,
            _localToken,
            _from,
            _to,
            _id,
            _amount,
            _extraData
        );

        // Lock tokens into bridge
        deposits[_localToken][_remoteToken][_id] += _amount;
        IERC1155(_localToken).safeTransferFrom(_from, address(this), _id, _amount, "");

        // Send calldata into L2
        MESSENGER.sendMessage(OTHER_BRIDGE, message, _minGasLimit);
        emit ERC1155BridgeInitiated(
            _localToken,
            _remoteToken,
            _from,
            _to,
            _id,
            _amount,
            _extraData
        );
    }
}
