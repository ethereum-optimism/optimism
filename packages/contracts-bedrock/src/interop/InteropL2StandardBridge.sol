// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2StandardBridge as LegacyL2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { InteropL2CrossDomainMessenger } from "src/interop/InteropL2CrossDomainMessenger.sol";
import { InteropConstants } from "src/interop/InteropConstants.sol";

import { IOptimismMintableERC20 } from "src/universal/IOptimismMintableERC20.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { ISemver } from "src/universal/ISemver.sol";

import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

/// @title InteropL2StandardBridge
/// @notice InteropL2StandardBridge is a replacement of the L2StandardBridge
///         predeploy that supports both L2-L2 and L2-L1 bridging.
contract InteropL2StandardBridge is ISemver {
    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    // StandardBridge: copied internals & updated dispatch/message spec

    /// @notice Interop enabled L2CrossDomainMessenger
    InteropL2CrossDomainMessenger public immutable MESSENGER;

    constructor(address payable _messenger) {
        MESSENGER = InteropL2CrossDomainMessenger(_messenger);
    }

    /// @notice emitted whenever ETH is bridged to a destination
    event ETHBridgeInitiated(
        bytes32 indexed destinationChain, address indexed from, address indexed to, uint256 amount, bytes extraData
    );

    /// @notice emitted whenever ETH is bridged from a source
    event ETHBridgeFinalized(
        bytes32 indexed sourceChain, address indexed from, address indexed to, uint256 amount, bytes extraData
    );

    /// @notice emitted whenever an ERC20 is bridged to a destination
    event ERC20BridgeInitiated(
        bytes32 indexed destinationChain,
        address indexed localToken,
        address indexed from,
        address remoteToken,
        address to,
        uint256 amount,
        bytes extraData
    );

    /// @notice emitted whenever an ERC20 is bridged from to a destination
    event ERC20BridgeFinalized(
        bytes32 indexed sourceChain,
        address indexed localToken,
        address indexed from,
        address remoteToken,
        address to,
        uint256 amount,
        bytes extraData
    );

    modifier onlyEOA() {
        require(!Address.isContract(msg.sender), "InteropL2StandardBridge: can only be called from an EOA");
        _;
    }

    modifier onlyOtherBridge() {
        require(
            msg.sender == address(MESSENGER) && MESSENGER.xDomainMessageSender() == address(this),
            "StandardBridge: function can only be called from the other bridge"
        );
        _;
    }

    /// @notice bridgeETHTo transfers ETH to the provided address on the recipient chain
    function bridgeETHTo(
        bytes32 targetChain,
        address _to,
        uint32 _minGasLimit,
        bytes calldata _extraData
    )
        public
        payable
        onlyEOA
    {

        // L2->L1 Support: Utilize the old pathway for now
        if (targetChain == InteropConstants.ETH_MAINNET_ID) {
            LegacyL2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeETHTo{ value: msg.value }(
                _to, _minGasLimit, _extraData
            );
            return;
        }

        emit ETHBridgeInitiated(targetChain, msg.sender, _to, msg.value, _extraData);
        MESSENGER.sendMessage{ value: msg.value }(
            targetChain,
            address(this),
            abi.encodeWithSelector(this.finalizeBridgeETH.selector, msg.sender, _to, msg.value, _extraData),
            _minGasLimit
        );
    }

    /// @notice bridgeERC20 burns and mints a native Optimism ERC20 to the provided address on
    //          the recipient chain. The token must be deployed on the specified target chain.
    function bridgeERC20To(
        bytes32 targetChain,
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    )
        public
        onlyEOA
    {
        require(
            ERC165Checker.supportsInterface(_localToken, type(IOptimismMintableERC20).interfaceId),
            "InteropL2StandardBridge: can only bridge the IOptimismMintableERC20 interface"
        );

        // L2->L1 Support: Utilize the old pathway for now
        if (targetChain == InteropConstants.ETH_MAINNET_ID) {
            LegacyL2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeERC20To(
                _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData
            );
            return;
        }

        require(
            _remoteToken == IOptimismMintableERC20(_localToken).remoteToken(),
            "InteropL2StandardBridge: wrong remote token for Optimism Mintable ERC20 local token"
        );

        IOptimismMintableERC20(_localToken).burn(msg.sender, _amount);
        emit ERC20BridgeInitiated(targetChain, _localToken, _remoteToken, msg.sender, _to, _amount, _extraData);

        MESSENGER.sendMessage(
            targetChain,
            address(this),
            // _localToken & _remoteToken stay in the same order cross-l2
            abi.encodeWithSelector(this.finalizeBridgeERC20.selector,
                _localToken, _remoteToken, msg.sender, _to, _amount, _extraData),
            _minGasLimit
        );
    }

    /// @notice finalizeBridgeETH releases ETH to the specified recipient
    function finalizeBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        public
        payable
        onlyOtherBridge
    {
        require(msg.value == _amount, "InteropL2StandardBridge: amount sent does not match amount required");
        require(_to != address(this), "InteropL2StandardBridge: cannot send to self");
        require(_to != address(MESSENGER), "InteropL2StandardBridge: cannot send to messenger");

        emit ETHBridgeFinalized(MESSENGER.xDomainChainId(), _from, _to, _amount, _extraData);

        bool success = SafeCall.call(_to, gasleft(), _amount, hex"");
        require(success, "InteropL2StandardBridge: ETH transfer failed");
    }

    /// @notice finalizeBridgeERC20 mints the amount of native Optimism ERC20 tokens to
    ///         the specified recipient. **NOTE** If this token does not exist, this
    ///         finalization step will fail until the token is deployed.
    function finalizeBridgeERC20(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        public
        onlyOtherBridge
    {
        //  NOTE: Do we want to sent a callback to unlock the tokens on the source chain
        //  if this local token does not exist? Funds are immediately burned on the source
        //  chain and "lost" until the token is deployed on this destination chain and
        //  this message replayed via the InteropL2CDM.

        require(
            _remoteToken == IOptimismMintableERC20(_localToken).remoteToken(),
            "InteropL2StandardBridge: wrong remote token for Optimism Mintable ERC20 local token"
        );

        IOptimismMintableERC20(_localToken).mint(_to, _amount);
        emit ERC20BridgeFinalized(
            MESSENGER.xDomainChainId(), _localToken, _from, _remoteToken, _to, _amount, _extraData
        );
    }
}
