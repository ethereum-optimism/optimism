// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { SuperchainWrappedERC20 } from "src/L2/SuperchainWrappedERC20.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { ZeroAddress, Unauthorized } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IERC20Metadata } from "@openzeppelin/contracts/token/ERC20/extensions/IERC20Metadata.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000026
/// @title SuperchainERC20Factory
/// @notice Deploys SuperchainWrappedERC20 tokens that wrap existing ERC20 tokens so they can move
///         across the Superchain through the SuperchainTokenBridge. Wrapped tokens are deployed
///         with CREATE2 using a salt derived solely from the (chainId, token) pair of the original
///         token, so a wrapped token has the same address on every chain. On the original token's
///         chain, this factory escrows original tokens 1:1 against minted wrapped tokens (wrap)
///         and releases them when wrapped tokens are burned (unwrap). Deployments are propagated
///         to other chains through the L2ToL2CrossDomainMessenger (`propagate`/`relayDeploy`), so
///         remote chains receive the canonical metadata without trusting a caller.
contract SuperchainERC20Factory {
    using SafeERC20 for IERC20;

    /// @notice Thrown when attempting to wrap or unwrap a token whose wrapped representation has
    ///         not been deployed yet.
    error SuperchainERC20Factory_TokenNotDeployed();

    /// @notice Thrown when reading the deploy configuration outside of a deployment.
    error SuperchainERC20Factory_NoActiveDeployment();

    /// @notice Thrown when relaying a deploy and the cross domain message sender is not the
    ///         SuperchainERC20Factory on the source chain.
    error SuperchainERC20Factory_InvalidCrossDomainSender();

    /// @notice Thrown when relaying a deploy whose message did not originate on the original
    ///         token's chain.
    error SuperchainERC20Factory_InvalidSourceChain();

    /// @notice Emitted when a new SuperchainWrappedERC20 is deployed.
    /// @param originalChainId Chain ID of the chain where the original token lives.
    /// @param originalToken   Address of the original token.
    /// @param wrappedToken    Address of the deployed wrapped token.
    /// @param name            Name of the wrapped token.
    /// @param symbol          Symbol of the wrapped token.
    /// @param decimals        Decimals of the wrapped token.
    event WrappedTokenDeployed(
        uint256 indexed originalChainId,
        address indexed originalToken,
        address indexed wrappedToken,
        string name,
        string symbol,
        uint8 decimals
    );

    /// @notice Emitted when a wrapped token's deployment is propagated to another chain.
    /// @param originalToken Address of the original token.
    /// @param wrappedToken  Address of the wrapped token.
    /// @param toChainId     Chain ID of the destination chain.
    /// @param msgHash       Hash of the message sent to the destination chain.
    event WrappedTokenPropagated(
        address indexed originalToken, address indexed wrappedToken, uint256 indexed toChainId, bytes32 msgHash
    );

    /// @notice Emitted when original tokens are wrapped.
    /// @param originalToken Address of the original token.
    /// @param wrappedToken  Address of the wrapped token.
    /// @param sender        Address that provided the original tokens.
    /// @param amount        Amount of wrapped tokens minted.
    event Wrapped(address indexed originalToken, address indexed wrappedToken, address indexed sender, uint256 amount);

    /// @notice Emitted when wrapped tokens are unwrapped back into original tokens.
    /// @param originalToken Address of the original token.
    /// @param wrappedToken  Address of the wrapped token.
    /// @param sender        Address that burned the wrapped tokens.
    /// @param amount        Amount of wrapped tokens burned.
    event Unwrapped(
        address indexed originalToken, address indexed wrappedToken, address indexed sender, uint256 amount
    );

    /// @notice Deployment parameters for the SuperchainWrappedERC20 currently being deployed.
    /// @dev Only set for the duration of a `deploy` call. The wrapped token's constructor reads
    ///      these back via `deployConfig()` so its creation code stays constant.
    struct DeployConfig {
        uint256 chainId;
        address token;
        string name;
        string symbol;
        uint8 decimals;
        bool active;
    }

    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    /// @notice Deploy configuration of the in-flight deployment.
    DeployConfig internal config;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the deployment parameters for the wrapped token currently being deployed.
    ///         Only callable during a `deploy` call (i.e. from the SuperchainWrappedERC20
    ///         constructor).
    /// @return chainId_  Chain ID of the chain where the original token lives.
    /// @return token_    Address of the original token.
    /// @return name_     Name of the wrapped token.
    /// @return symbol_   Symbol of the wrapped token.
    /// @return decimals_ Decimals of the wrapped token.
    function deployConfig()
        external
        view
        returns (uint256 chainId_, address token_, string memory name_, string memory symbol_, uint8 decimals_)
    {
        if (!config.active) revert SuperchainERC20Factory_NoActiveDeployment();
        return (config.chainId, config.token, config.name, config.symbol, config.decimals);
    }

    /// @notice Deploys the SuperchainWrappedERC20 for the given (chainId, token) pair. The
    ///         resulting address depends only on that pair, so deploying on multiple chains yields
    ///         the same address everywhere. If the original token lives on this chain, its
    ///         metadata is read directly from the token where possible; otherwise the supplied
    ///         metadata is used.
    /// @param _chainId  Chain ID of the chain where the original token lives.
    /// @param _token    Address of the original token.
    /// @param _name     Name for the wrapped token. Ignored when the original token is on this
    ///                  chain and exposes `name()`.
    /// @param _symbol   Symbol for the wrapped token. Ignored when the original token is on this
    ///                  chain and exposes `symbol()`.
    /// @param _decimals Decimals for the wrapped token. Ignored when the original token is on this
    ///                  chain and exposes `decimals()`.
    /// @return wrappedToken_ Address of the deployed wrapped token.
    function deploy(
        uint256 _chainId,
        address _token,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        returns (address wrappedToken_)
    {
        if (_token == address(0)) revert ZeroAddress();

        // On the original token's chain the canonical metadata is available on the token itself,
        // so prefer it over the caller-supplied values. Tokens that do not implement the ERC20
        // metadata extension fall back to the caller-supplied values.
        if (_chainId == block.chainid) {
            try IERC20Metadata(_token).name() returns (string memory name_) {
                _name = name_;
            } catch { }
            try IERC20Metadata(_token).symbol() returns (string memory symbol_) {
                _symbol = symbol_;
            } catch { }
            try IERC20Metadata(_token).decimals() returns (uint8 decimals_) {
                _decimals = decimals_;
            } catch { }
        }

        wrappedToken_ = _deploy(_chainId, _token, _name, _symbol, _decimals);
    }

    /// @notice Propagates a wrapped token's deployment to another chain via the
    ///         L2ToL2CrossDomainMessenger. Only possible on the original token's chain, and only
    ///         after the wrapped token has been deployed there, so the propagated metadata always
    ///         matches the canonical wrapped token. On the destination chain the factory deploys
    ///         the wrapped token at the same address through `relayDeploy`.
    /// @param _token     Address of the original token whose wrapped token should be propagated.
    /// @param _toChainId Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent to the destination chain.
    function propagate(address _token, uint256 _toChainId) external returns (bytes32 msgHash_) {
        address wrappedToken = _deployedWrappedToken(_token);

        bytes memory message = abi.encodeCall(
            this.relayDeploy,
            (
                block.chainid,
                _token,
                IERC20Metadata(wrappedToken).name(),
                IERC20Metadata(wrappedToken).symbol(),
                IERC20Metadata(wrappedToken).decimals()
            )
        );
        msgHash_ = IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(_toChainId, address(this), message);

        emit WrappedTokenPropagated(_token, wrappedToken, _toChainId, msgHash_);
    }

    /// @notice Deploys a wrapped token propagated from its original token's chain. Because the
    ///         factory only ever propagates tokens that live on its own chain, requiring the
    ///         message's source chain to equal `_chainId` guarantees the metadata was read from
    ///         the canonical wrapped token on the original token's chain.
    /// @dev If the wrapped token was already deployed on this chain (e.g. manually via `deploy`),
    ///      the CREATE2 deployment reverts and the message simply remains unrelayable, which is
    ///      harmless since the address is deterministic.
    /// @param _chainId  Chain ID of the chain where the original token lives.
    /// @param _token    Address of the original token.
    /// @param _name     Name of the wrapped token on the source chain.
    /// @param _symbol   Symbol of the wrapped token on the source chain.
    /// @param _decimals Decimals of the wrapped token on the source chain.
    /// @return wrappedToken_ Address of the deployed wrapped token.
    function relayDeploy(
        uint256 _chainId,
        address _token,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        returns (address wrappedToken_)
    {
        if (msg.sender != MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 source) =
            IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != address(this)) revert SuperchainERC20Factory_InvalidCrossDomainSender();
        if (source != _chainId) revert SuperchainERC20Factory_InvalidSourceChain();

        wrappedToken_ = _deploy(_chainId, _token, _name, _symbol, _decimals);
    }

    /// @notice Deploys the SuperchainWrappedERC20 for the given (chainId, token) pair with the
    ///         given metadata, exposing the parameters to the wrapped token's constructor via
    ///         `deployConfig()`.
    function _deploy(
        uint256 _chainId,
        address _token,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        internal
        returns (address wrappedToken_)
    {
        config = DeployConfig({
            chainId: _chainId,
            token: _token,
            name: _name,
            symbol: _symbol,
            decimals: _decimals,
            active: true
        });

        // Reverts if the wrapped token was already deployed, since CREATE2 cannot redeploy to the
        // same address.
        wrappedToken_ = address(new SuperchainWrappedERC20{ salt: _salt(_chainId, _token) }());

        delete config;

        emit WrappedTokenDeployed(_chainId, _token, wrappedToken_, _name, _symbol, _decimals);
    }

    /// @notice Wraps original tokens into their SuperchainWrappedERC20 representation. Only
    ///         possible on the original token's chain: the factory escrows the original tokens and
    ///         mints the same amount of wrapped tokens to the caller. The wrapped token must have
    ///         been deployed first via `deploy`.
    /// @param _token  Address of the original token to wrap.
    /// @param _amount Amount of original tokens to wrap.
    /// @return wrappedToken_ Address of the wrapped token that was minted.
    function wrap(address _token, uint256 _amount) external returns (address wrappedToken_) {
        wrappedToken_ = _deployedWrappedToken(_token);

        // Mint against the actually received amount so tokens that take fees on transfer cannot
        // inflate the wrapped supply beyond the escrowed backing.
        uint256 balanceBefore = IERC20(_token).balanceOf(address(this));
        IERC20(_token).safeTransferFrom(msg.sender, address(this), _amount);
        uint256 received = IERC20(_token).balanceOf(address(this)) - balanceBefore;

        SuperchainWrappedERC20(wrappedToken_).mint(msg.sender, received);

        emit Wrapped(_token, wrappedToken_, msg.sender, received);
    }

    /// @notice Unwraps SuperchainWrappedERC20 tokens back into the original tokens. Only possible
    ///         on the original token's chain: the factory burns the caller's wrapped tokens and
    ///         releases the same amount of escrowed original tokens.
    /// @param _token  Address of the original token to unwrap into.
    /// @param _amount Amount of wrapped tokens to unwrap.
    /// @return wrappedToken_ Address of the wrapped token that was burned.
    function unwrap(address _token, uint256 _amount) external returns (address wrappedToken_) {
        wrappedToken_ = _deployedWrappedToken(_token);

        SuperchainWrappedERC20(wrappedToken_).burn(msg.sender, _amount);

        IERC20(_token).safeTransfer(msg.sender, _amount);

        emit Unwrapped(_token, wrappedToken_, msg.sender, _amount);
    }

    /// @notice Computes the address of the SuperchainWrappedERC20 for the given (chainId, token)
    ///         pair. The address is the same on every chain, whether or not the wrapped token has
    ///         been deployed yet.
    /// @param _chainId Chain ID of the chain where the original token lives.
    /// @param _token   Address of the original token.
    /// @return wrappedToken_ Address of the wrapped token.
    function getWrappedToken(uint256 _chainId, address _token) public view returns (address wrappedToken_) {
        bytes32 hash = keccak256(
            abi.encodePacked(
                bytes1(0xff),
                address(this),
                _salt(_chainId, _token),
                keccak256(type(SuperchainWrappedERC20).creationCode)
            )
        );
        wrappedToken_ = address(uint160(uint256(hash)));
    }

    /// @notice Returns the wrapped token for an original token on this chain, reverting if it has
    ///         not been deployed yet.
    /// @param _token Address of the original token.
    /// @return wrappedToken_ Address of the deployed wrapped token.
    function _deployedWrappedToken(address _token) internal view returns (address wrappedToken_) {
        wrappedToken_ = getWrappedToken(block.chainid, _token);
        if (wrappedToken_.code.length == 0) revert SuperchainERC20Factory_TokenNotDeployed();
    }

    /// @notice Computes the CREATE2 salt for the given (chainId, token) pair.
    /// @param _chainId Chain ID of the chain where the original token lives.
    /// @param _token   Address of the original token.
    /// @return salt_ The CREATE2 salt.
    function _salt(uint256 _chainId, address _token) internal pure returns (bytes32 salt_) {
        salt_ = keccak256(abi.encode(_chainId, _token));
    }
}
