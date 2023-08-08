// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { StandardBridge } from "./StandardBridge.sol";

abstract contract StandardBridgeExtension is StandardBridge {
    address public immutable LOCAL_TOKEN;
    address public immutable REMOTE_TOKEN;

    constructor(
        address payable _messenger,
        address payable _otherBridge,
        address _localToken,
        address _remoteToken
    ) StandardBridge(_messenger, _otherBridge) {
        LOCAL_TOKEN = _localToken;
        REMOTE_TOKEN = _remoteToken;
    }

    modifier onlyERC20(address _localToken, address remoteToken) {
        require(
            LOCAL_TOKEN == _localToken && REMOTE_TOKEN == remoteToken,
            "ERC20StandardBridge: can only bridge specified ERC20"
        );
        _;
    }

    /**
     * Disable ETH Bridging
     *   - (ideally we could re-route this to the official StandardBridge via the right address from SytemConfig)
     */

    function _initiateBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    ) internal pure override {
        revert("StandardBridgeExtensions: cannot bridge ETH using a standard bridge extension");
    }

    receive() external override payable {
        revert("StandardBridgeExtensions: cannot bridge ETH using a standard bridge extension");
    }

    /**
     * Bridge only the registered ERC20
     */

    /// @notice Bridge the registered ERC20 tokeen to the sender's address
    function bridge(uint256 _amount, uint32 _minGasLimit, bytes calldata _extraData) public virtual {
        bridgeERC20(LOCAL_TOKEN, REMOTE_TOKEN, _amount, _minGasLimit, _extraData);
    }

    /// @notice Bridge the registered ERC20 tokeen to the specified receiver's address.
    function bridgeTo(address _to, uint256 _amount, uint32 _minGasLimit, bytes calldata _extraData) public virtual {
        bridgeERC20To(LOCAL_TOKEN, REMOTE_TOKEN, _to, _amount, _minGasLimit, _extraData);
    }

    function bridgeERC20(
        address _localToken,
        address _remoteToken,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public override onlyERC20(_localToken, _remoteToken) {
        super.bridgeERC20(_localToken, _remoteToken, _amount, _minGasLimit, _extraData);
    }

    function bridgeERC20To(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public override onlyERC20(_localToken, _remoteToken) {
        super.bridgeERC20To(_localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData);
    }
}
