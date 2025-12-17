// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

// Target contract
import { TransferOnion } from "src/periphery/TransferOnion.sol";

/// @title TransferOnion_TestInit
/// @notice Reusable test initialization for `TransferOnion` tests.
abstract contract TransferOnion_TestInit is Test {
    /// @notice TransferOnion
    TransferOnion internal onion;

    /// @notice Token constructor argument
    address internal _token;

    /// @notice Sender constructor argument
    address internal _sender;

    /// @notice Sets up addresses, deploys contracts and funds the owner.
    function setUp() public {
        ERC20 token = new ERC20("Token", "TKN");
        _token = address(token);
        _sender = makeAddr("sender");
    }

    /// @notice Deploy the TransferOnion with a dummy shell.
    function _deploy() public {
        _deploy(bytes32(0));
    }

    /// @notice Deploy the TransferOnion with a specific shell.
    function _deploy(bytes32 _shell) public {
        onion = new TransferOnion({ _token: ERC20(_token), _sender: _sender, _shell: _shell });
    }

    /// @notice Build the onion data.
    function _onionize(TransferOnion.Layer[] memory _layers)
        public
        pure
        returns (bytes32, TransferOnion.Layer[] memory)
    {
        uint256 length = _layers.length;
        bytes32 hash = bytes32(0);
        for (uint256 i; i < length; i++) {
            TransferOnion.Layer memory layer = _layers[i];
            _layers[i].shell = hash;
            hash = keccak256(abi.encode(layer.recipient, layer.amount, hash));
        }
        return (hash, _layers);
    }
}

/// @title TransferOnion_Constructor_Test
/// @notice Tests the constructor of the `TransferOnion` contract.
contract TransferOnion_Constructor_Test is TransferOnion_TestInit {
    /// @notice The constructor sets the variables as expected.
    function test_constructor_succeeds() external {
        _deploy();

        assertEq(address(onion.TOKEN()), _token);
        assertEq(onion.SENDER(), _sender);
        assertEq(onion.shell(), bytes32(0));
    }
}

/// @title TransferOnion_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `TransferOnion`
///         contract or are testing multiple functions at once.
contract TransferOnion_Uncategorized_Test is TransferOnion_TestInit {
    /// @notice Tests unwrapping the onion.
    function test_unwrap_succeeds() external {
        // Commit to transferring tiny amounts of tokens
        TransferOnion.Layer[] memory _layers = new TransferOnion.Layer[](2);
        _layers[0] = TransferOnion.Layer(address(1), 1, bytes32(0));
        _layers[1] = TransferOnion.Layer(address(2), 2, bytes32(0));

        // Build the onion shell
        (bytes32 shell, TransferOnion.Layer[] memory layers) = _onionize(_layers);
        _deploy(shell);

        assertEq(onion.shell(), shell);

        address token = address(onion.TOKEN());
        address sender = onion.SENDER();

        // give 3 units of token to sender
        deal(token, onion.SENDER(), 3);
        vm.prank(sender);
        ERC20(token).approve(address(onion), 3);

        // To build the inputs, to `peel`, need to reverse the list
        TransferOnion.Layer[] memory inputs = new TransferOnion.Layer[](2);
        int256 length = int256(layers.length);
        for (int256 i = length - 1; i >= 0; i--) {
            uint256 ui = uint256(i);
            uint256 revidx = uint256(length) - ui - 1;
            TransferOnion.Layer memory layer = layers[ui];
            inputs[revidx] = layer;
        }

        // The accounts have no balance
        assertEq(ERC20(_token).balanceOf(address(1)), 0);
        assertEq(ERC20(_token).balanceOf(address(2)), 0);

        onion.peel(inputs);

        // Now the accounts have the expected balance
        assertEq(ERC20(_token).balanceOf(address(1)), 1);
        assertEq(ERC20(_token).balanceOf(address(2)), 2);
    }
}
