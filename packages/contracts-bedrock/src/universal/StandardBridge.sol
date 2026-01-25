// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Libraries
import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { EOA } from "src/libraries/EOA.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IOptimismMintableERC20 } from "interfaces/universal/IOptimismMintableERC20.sol";
import { ILegacyMintableERC20 } from "interfaces/legacy/ILegacyMintableERC20.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";

/**
 * @custom:upgradeable
 * @title StandardBridge
 * @notice StandardBridge is a base contract for the L1 and L2 standard ERC20 bridges.
 */
abstract contract StandardBridge is Initializable {
    using SafeERC20 for IERC20;

    // --- Custom Errors ---
    error StandardBridge_Paused();
    error StandardBridge_OnlyOtherBridge();
    error StandardBridge_OnlyEOA();
    error StandardBridge_InvalidAmount();
    error StandardBridge_InvalidRecipient();
    error StandardBridge_TransferFailed();
    error StandardBridge_InvalidTokenPair();

    uint32 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 200_000;

    bytes30 private spacer_0_2_30;
    address private spacer_1_0_20;

    mapping(address => mapping(address => uint256)) public deposits;

    ICrossDomainMessenger public messenger;
    StandardBridge public otherBridge;

    uint256[45] private __gap;

    event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData);
    event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData);
    event ERC20BridgeInitiated(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 amount,
        bytes extraData
    );
    event ERC20BridgeFinalized(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 amount,
        bytes extraData
    );

    modifier onlyEOA() {
        if (!EOA.isSenderEOA()) revert StandardBridge_OnlyEOA();
        _;
    }

    modifier onlyOtherBridge() {
        if (msg.sender != address(messenger) || messenger.xDomainMessageSender() != address(otherBridge)) {
            revert StandardBridge_OnlyOtherBridge();
        }
        _;
    }

    function __StandardBridge_init(
        ICrossDomainMessenger _messenger,
        StandardBridge _otherBridge
    ) internal onlyInitializing {
        messenger = _messenger;
        otherBridge = _otherBridge;
    }

    receive() external payable virtual;

    function MESSENGER() external view returns (ICrossDomainMessenger) { return messenger; }
    function OTHER_BRIDGE() external view returns (StandardBridge) { return otherBridge; }
    function paused() public view virtual returns (bool) { return false; }

    function bridgeETH(uint32 _minGasLimit, bytes calldata _extraData) public payable onlyEOA {
        _initiateBridgeETH(msg.sender, msg.sender, msg.value, _minGasLimit, _extraData);
    }

    function bridgeETHTo(address _to, uint32 _minGasLimit, bytes calldata _extraData) public payable {
        _initiateBridgeETH(msg.sender, _to, msg.value, _minGasLimit, _extraData);
    }

    function bridgeERC20(
        address _localToken,
        address _remoteToken,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public virtual onlyEOA {
        _initiateBridgeERC20(_localToken, _remoteToken, msg.sender, msg.sender, _amount, _minGasLimit, _extraData);
    }

    function bridgeERC20To(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public virtual {
        _initiateBridgeERC20(_localToken, _remoteToken, msg.sender, _to, _amount, _minGasLimit, _extraData);
    }

    function finalizeBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    ) public payable onlyOtherBridge {
        if (paused()) revert StandardBridge_Paused();
        if (msg.value != _amount) revert StandardBridge_InvalidAmount();
        if (_to == address(this) || _to == address(messenger)) revert StandardBridge_InvalidRecipient();

        _emitETHBridgeFinalized(_from, _to, _amount, _extraData);

        if (!SafeCall.call(_to, gasleft(), _amount, hex"")) revert StandardBridge_TransferFailed();
    }

    function finalizeBridgeERC20(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    ) public onlyOtherBridge {
        if (paused()) revert StandardBridge_Paused();
        
        if (_isOptimismMintableERC20(_localToken)) {
            if (!_isCorrectTokenPair(_localToken, _remoteToken)) revert StandardBridge_InvalidTokenPair();
            IOptimismMintableERC20(_localToken).mint(_to, _amount);
        } else {
            unchecked {
                deposits[_localToken][_remoteToken] -= _amount;
            }
            IERC20(_localToken).safeTransfer(_to, _amount);
        }

        _emitERC20BridgeFinalized(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }

    function _initiateBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    ) internal {
        if (msg.value != _amount) revert StandardBridge_InvalidAmount();

        _emitETHBridgeInitiated(_from, _to, _amount, _extraData);

        messenger.sendMessage{ value: _amount }({
            _target: address(otherBridge),
            _message: abi.encodeWithSelector(this.finalizeBridgeETH.selector, _from, _to, _amount, _extraData),
            _minGasLimit: _minGasLimit
        });
    }

    function _initiateBridgeERC20(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    ) internal {
        if (msg.value != 0) revert StandardBridge_InvalidAmount();

        if (_isOptimismMintableERC20(_localToken)) {
            if (!_isCorrectTokenPair(_localToken, _remoteToken)) revert StandardBridge_InvalidTokenPair();
            IOptimismMintableERC20(_localToken).burn(_from, _amount);
        } else {
            IERC20(_localToken).safeTransferFrom(_from, address(this), _amount);
            unchecked {
                deposits[_localToken][_remoteToken] += _amount;
            }
        }

        _emitERC20BridgeInitiated(_localToken, _remoteToken, _from, _to, _amount, _extraData);

        messenger.sendMessage({
            _target: address(otherBridge),
            _message: abi.encodeWithSelector(
                this.finalizeBridgeERC20.selector,
                _remoteToken,
                _localToken,
                _from,
                _to,
                _amount,
                _extraData
            ),
            _minGasLimit: _minGasLimit
        });
    }

    function _isOptimismMintableERC20(address _token) internal view returns (bool) {
        return ERC165Checker.supportsInterface(_token, type(ILegacyMintableERC20).interfaceId)
            || ERC165Checker.supportsInterface(_token, type(IOptimismMintableERC20).interfaceId);
    }

    function _isCorrectTokenPair(address _mintableToken, address _otherToken) internal view returns (bool) {
        if (ERC165Checker.supportsInterface(_mintableToken, type(ILegacyMintableERC20).interfaceId)) {
            return _otherToken == ILegacyMintableERC20(_mintableToken).l1Token();
        } else {
            return _otherToken == IOptimismMintableERC20(_mintableToken).remoteToken();
        }
    }

    function _emitETHBridgeInitiated(address _from, address _to, uint256 _amount, bytes memory _extraData) internal virtual {
        emit ETHBridgeInitiated(_from, _to, _amount, _extraData);
    }

    function _emitETHBridgeFinalized(address _from, address _to, uint256 _amount, bytes memory _extraData) internal virtual {
        emit ETHBridgeFinalized(_from, _to, _amount, _extraData);
    }

    function _emitERC20BridgeInitiated(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes memory _extraData) internal virtual {
        emit ERC20BridgeInitiated(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }

    function _emitERC20BridgeFinalized(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes memory _extraData) internal virtual {
        emit ERC20BridgeFinalized(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }
}
