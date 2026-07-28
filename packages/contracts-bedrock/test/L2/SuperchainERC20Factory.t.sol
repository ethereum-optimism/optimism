// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing
import { Test } from "test/setup/Test.sol";
import { MockHelper } from "test/utils/MockHelper.sol";
import { TestERC20, FeeOnTransferERC20, NoMetadataERC20 } from "test/mocks/TestTokens.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

// Target contract
import { SuperchainERC20Factory } from "src/L2/SuperchainERC20Factory.sol";
import { ISuperchainERC20Factory } from "interfaces/L2/ISuperchainERC20Factory.sol";
import { ISuperchainWrappedERC20 } from "interfaces/L2/ISuperchainWrappedERC20.sol";

/// @title SuperchainERC20Factory_TestInit
/// @notice Reusable test initialization for `SuperchainERC20Factory` tests.
abstract contract SuperchainERC20Factory_TestInit is Test, MockHelper {
    event WrappedTokenDeployed(
        uint256 indexed originalChainId,
        address indexed originalToken,
        address indexed wrappedToken,
        string name,
        string symbol,
        uint8 decimals
    );

    event WrappedTokenPropagated(
        address indexed originalToken, address indexed wrappedToken, uint256 indexed toChainId, bytes32 msgHash
    );

    event Wrapped(address indexed originalToken, address indexed wrappedToken, address indexed sender, uint256 amount);

    event Unwrapped(
        address indexed originalToken, address indexed wrappedToken, address indexed sender, uint256 amount
    );

    uint256 internal constant REMOTE_CHAIN_ID = 901;

    ISuperchainERC20Factory public factory;
    TestERC20 public token;
    address public alice;

    /// @notice Sets up the test suite.
    function setUp() public virtual {
        vm.etch(Predeploys.SUPERCHAIN_ERC20_FACTORY, address(new SuperchainERC20Factory()).code);
        factory = ISuperchainERC20Factory(Predeploys.SUPERCHAIN_ERC20_FACTORY);
        token = new TestERC20("Test Token", "TEST", 18);
        alice = makeAddr("alice");
    }

    /// @notice Deploys the wrapped token for `token` on the current chain and funds `alice`.
    function _deployWrappedToken() internal returns (ISuperchainWrappedERC20 wrapped_) {
        wrapped_ = ISuperchainWrappedERC20(factory.deploy(block.chainid, address(token), "", "", 0));
        token.mint(alice, 1000 ether);
        vm.prank(alice);
        token.approve(address(factory), type(uint256).max);
    }
}

/// @title SuperchainERC20Factory_Deploy_Test
/// @notice Tests the `deploy` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_Deploy_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the `deploy` function reverts when the token is the zero address.
    function testFuzz_deploy_zeroAddressToken_reverts(uint256 _chainId) public {
        vm.expectRevert(ISuperchainERC20Factory.ZeroAddress.selector);
        factory.deploy(_chainId, address(0), "", "", 0);
    }

    /// @notice Tests the `deploy` function reads the metadata from the token when the original
    ///         token lives on this chain, ignoring the provided values.
    function test_deploy_homeChainReadsMetadata_succeeds() public {
        address predicted = factory.getWrappedToken(block.chainid, address(token));

        vm.expectEmit(address(factory));
        emit WrappedTokenDeployed(block.chainid, address(token), predicted, "Test Token", "TEST", 18);

        address wrapped = factory.deploy(block.chainid, address(token), "ignored", "IGN", 6);

        assertEq(wrapped, predicted);
        assertEq(ISuperchainWrappedERC20(wrapped).name(), "Test Token");
        assertEq(ISuperchainWrappedERC20(wrapped).symbol(), "TEST");
        assertEq(ISuperchainWrappedERC20(wrapped).decimals(), 18);
        assertEq(ISuperchainWrappedERC20(wrapped).FACTORY(), address(factory));
        assertEq(ISuperchainWrappedERC20(wrapped).ORIGINAL_TOKEN(), address(token));
        assertEq(ISuperchainWrappedERC20(wrapped).ORIGINAL_CHAIN_ID(), block.chainid);
    }

    /// @notice Tests the `deploy` function uses the provided metadata when the original token
    ///         lives on another chain.
    function test_deploy_remoteChainUsesProvidedMetadata_succeeds() public {
        address wrapped = factory.deploy(REMOTE_CHAIN_ID, address(token), "Remote Token", "RMT", 6);

        assertEq(ISuperchainWrappedERC20(wrapped).name(), "Remote Token");
        assertEq(ISuperchainWrappedERC20(wrapped).symbol(), "RMT");
        assertEq(ISuperchainWrappedERC20(wrapped).decimals(), 6);
        assertEq(ISuperchainWrappedERC20(wrapped).ORIGINAL_CHAIN_ID(), REMOTE_CHAIN_ID);
    }

    /// @notice Tests the `deploy` function falls back to the provided metadata when the original
    ///         token lives on this chain but does not expose the metadata extension.
    function test_deploy_homeChainNoMetadata_succeeds() public {
        NoMetadataERC20 rawToken = new NoMetadataERC20();

        address wrapped = factory.deploy(block.chainid, address(rawToken), "Raw Token", "RAW", 8);

        assertEq(ISuperchainWrappedERC20(wrapped).name(), "Raw Token");
        assertEq(ISuperchainWrappedERC20(wrapped).symbol(), "RAW");
        assertEq(ISuperchainWrappedERC20(wrapped).decimals(), 8);
    }

    /// @notice Tests the `deploy` function reverts when the wrapped token was already deployed.
    function test_deploy_alreadyDeployed_reverts() public {
        factory.deploy(block.chainid, address(token), "", "", 0);

        vm.expectRevert();
        factory.deploy(block.chainid, address(token), "", "", 0);
    }
}

/// @title SuperchainERC20Factory_DeployConfig_Test
/// @notice Tests the `deployConfig` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_DeployConfig_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the `deployConfig` function reverts outside of a deployment.
    function test_deployConfig_noActiveDeployment_reverts() public {
        vm.expectRevert(ISuperchainERC20Factory.SuperchainERC20Factory_NoActiveDeployment.selector);
        factory.deployConfig();
    }
}

/// @title SuperchainERC20Factory_GetWrappedToken_Test
/// @notice Tests the `getWrappedToken` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_GetWrappedToken_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the wrapped token address depends only on the (chainId, token) pair and not
    ///         on the provided metadata.
    function testFuzz_getWrappedToken_deterministicAddress_succeeds(
        uint256 _chainId,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        public
    {
        vm.assume(_chainId != block.chainid);

        address predicted = factory.getWrappedToken(_chainId, address(token));
        address wrapped = factory.deploy(_chainId, address(token), _name, _symbol, _decimals);

        assertEq(wrapped, predicted);
    }
}

/// @title SuperchainERC20Factory_Propagate_Test
/// @notice Tests the `propagate` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_Propagate_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the `propagate` function reverts when the wrapped token has not been
    ///         deployed on this chain.
    function test_propagate_notDeployed_reverts() public {
        vm.expectRevert(ISuperchainERC20Factory.SuperchainERC20Factory_TokenNotDeployed.selector);
        factory.propagate(address(token), REMOTE_CHAIN_ID);
    }

    /// @notice Tests the `propagate` function sends a `relayDeploy` message with the canonical
    ///         wrapped token metadata to the factory on the destination chain and emits the
    ///         `WrappedTokenPropagated` event.
    function testFuzz_propagate_succeeds(uint256 _toChainId, bytes32 _msgHash) public {
        ISuperchainWrappedERC20 wrapped = _deployWrappedToken();

        // Mock the call over the `sendMessage` function and expect it to be called with a
        // `relayDeploy` message carrying the wrapped token's metadata.
        bytes memory message = abi.encodeCall(
            ISuperchainERC20Factory.relayDeploy, (block.chainid, address(token), "Test Token", "TEST", 18)
        );
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sendMessage, (_toChainId, address(factory), message)),
            abi.encode(_msgHash)
        );

        vm.expectEmit(address(factory));
        emit WrappedTokenPropagated(address(token), address(wrapped), _toChainId, _msgHash);

        bytes32 msgHash = factory.propagate(address(token), _toChainId);

        assertEq(msgHash, _msgHash);
    }
}

/// @title SuperchainERC20Factory_RelayDeploy_Test
/// @notice Tests the `relayDeploy` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_RelayDeploy_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the `relayDeploy` function reverts when the caller is not the
    ///         `L2ToL2CrossDomainMessenger`.
    function testFuzz_relayDeploy_notMessenger_reverts(address _caller) public {
        vm.assume(_caller != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        vm.expectRevert(ISuperchainERC20Factory.Unauthorized.selector);
        vm.prank(_caller);
        factory.relayDeploy(REMOTE_CHAIN_ID, address(token), "Test Token", "TEST", 18);
    }

    /// @notice Tests the `relayDeploy` function reverts when the cross domain message sender is
    ///         not the SuperchainERC20Factory.
    function testFuzz_relayDeploy_notCrossDomainSender_reverts(address _crossDomainMessageSender) public {
        vm.assume(_crossDomainMessageSender != address(factory));

        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(_crossDomainMessageSender, REMOTE_CHAIN_ID)
        );

        vm.expectRevert(ISuperchainERC20Factory.SuperchainERC20Factory_InvalidCrossDomainSender.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        factory.relayDeploy(REMOTE_CHAIN_ID, address(token), "Test Token", "TEST", 18);
    }

    /// @notice Tests the `relayDeploy` function reverts when the message did not originate on the
    ///         chain the original token is claimed to live on.
    function testFuzz_relayDeploy_wrongSourceChain_reverts(uint256 _source) public {
        vm.assume(_source != REMOTE_CHAIN_ID);

        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(factory), _source)
        );

        vm.expectRevert(ISuperchainERC20Factory.SuperchainERC20Factory_InvalidSourceChain.selector);
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        factory.relayDeploy(REMOTE_CHAIN_ID, address(token), "Test Token", "TEST", 18);
    }

    /// @notice Tests the `relayDeploy` function deploys the wrapped token at the deterministic
    ///         address with the propagated metadata.
    function test_relayDeploy_succeeds() public {
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(factory), REMOTE_CHAIN_ID)
        );

        address predicted = factory.getWrappedToken(REMOTE_CHAIN_ID, address(token));

        vm.expectEmit(address(factory));
        emit WrappedTokenDeployed(REMOTE_CHAIN_ID, address(token), predicted, "Test Token", "TEST", 18);

        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        address wrapped = factory.relayDeploy(REMOTE_CHAIN_ID, address(token), "Test Token", "TEST", 18);

        assertEq(wrapped, predicted);
        assertEq(ISuperchainWrappedERC20(wrapped).name(), "Test Token");
        assertEq(ISuperchainWrappedERC20(wrapped).symbol(), "TEST");
        assertEq(ISuperchainWrappedERC20(wrapped).decimals(), 18);
        assertEq(ISuperchainWrappedERC20(wrapped).ORIGINAL_CHAIN_ID(), REMOTE_CHAIN_ID);
        assertEq(ISuperchainWrappedERC20(wrapped).ORIGINAL_TOKEN(), address(token));
    }
}

/// @title SuperchainERC20Factory_Wrap_Test
/// @notice Tests the `wrap` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_Wrap_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the `wrap` function reverts when the wrapped token has not been deployed.
    function test_wrap_notDeployed_reverts() public {
        vm.expectRevert(ISuperchainERC20Factory.SuperchainERC20Factory_TokenNotDeployed.selector);
        factory.wrap(address(token), 100);
    }

    /// @notice Tests the `wrap` function escrows the original tokens and mints the same amount of
    ///         wrapped tokens to the caller.
    function testFuzz_wrap_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, 1000 ether);
        ISuperchainWrappedERC20 wrapped = _deployWrappedToken();

        vm.expectEmit(address(factory));
        emit Wrapped(address(token), address(wrapped), alice, _amount);

        vm.prank(alice);
        factory.wrap(address(token), _amount);

        assertEq(wrapped.balanceOf(alice), _amount);
        assertEq(wrapped.totalSupply(), _amount);
        assertEq(token.balanceOf(alice), 1000 ether - _amount);
        assertEq(token.balanceOf(address(factory)), _amount);
    }

    /// @notice Tests the `wrap` function mints only the received amount for tokens that take a
    ///         fee on transfer.
    function test_wrap_feeOnTransferMintsReceivedAmount_succeeds() public {
        FeeOnTransferERC20 feeToken = new FeeOnTransferERC20();
        address wrapped = factory.deploy(block.chainid, address(feeToken), "", "", 0);
        feeToken.mint(alice, 100 ether);
        vm.prank(alice);
        feeToken.approve(address(factory), type(uint256).max);

        vm.prank(alice);
        factory.wrap(address(feeToken), 100 ether);

        uint256 received = 100 ether - (100 ether / 100);
        assertEq(ISuperchainWrappedERC20(wrapped).balanceOf(alice), received);
        assertEq(feeToken.balanceOf(address(factory)), received);
    }
}

/// @title SuperchainERC20Factory_Unwrap_Test
/// @notice Tests the `unwrap` function of the `SuperchainERC20Factory` contract.
contract SuperchainERC20Factory_Unwrap_Test is SuperchainERC20Factory_TestInit {
    /// @notice Tests the `unwrap` function reverts when the wrapped token has not been deployed.
    function test_unwrap_notDeployed_reverts() public {
        vm.expectRevert(ISuperchainERC20Factory.SuperchainERC20Factory_TokenNotDeployed.selector);
        factory.unwrap(address(token), 100);
    }

    /// @notice Tests the `unwrap` function burns the wrapped tokens and releases the escrowed
    ///         original tokens.
    function testFuzz_unwrap_succeeds(uint256 _wrapAmount, uint256 _unwrapAmount) public {
        _wrapAmount = bound(_wrapAmount, 0, 1000 ether);
        _unwrapAmount = bound(_unwrapAmount, 0, _wrapAmount);
        ISuperchainWrappedERC20 wrapped = _deployWrappedToken();

        vm.prank(alice);
        factory.wrap(address(token), _wrapAmount);

        vm.expectEmit(address(factory));
        emit Unwrapped(address(token), address(wrapped), alice, _unwrapAmount);

        vm.prank(alice);
        factory.unwrap(address(token), _unwrapAmount);

        assertEq(wrapped.balanceOf(alice), _wrapAmount - _unwrapAmount);
        assertEq(token.balanceOf(alice), 1000 ether - _wrapAmount + _unwrapAmount);
        assertEq(token.balanceOf(address(factory)), _wrapAmount - _unwrapAmount);
    }

    /// @notice Tests the `unwrap` function reverts when the caller has fewer wrapped tokens than
    ///         the requested amount.
    function test_unwrap_insufficientBalance_reverts() public {
        _deployWrappedToken();

        vm.prank(alice);
        factory.wrap(address(token), 100 ether);

        vm.expectRevert();
        vm.prank(alice);
        factory.unwrap(address(token), 100 ether + 1);
    }
}
