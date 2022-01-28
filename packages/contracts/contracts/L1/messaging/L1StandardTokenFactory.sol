// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Contract Imports */
import { L1StandardERC20 } from "../../standards/L1StandardERC20.sol";
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/**
 * @title L1StandardTokenFactory
 * @dev Factory contract for creating standard L1 token representations of L2 ERC20s
 * compatible with and working on the standard bridge.
 */
contract L1StandardTokenFactory {
    event StandardL1TokenCreated(address indexed _l2Token, address indexed _l1Token);

    /**
     * @dev Creates an instance of the standard ERC20 token on L1.
     * @param _l2Token Address of the corresponding L2 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    function createStandardL1Token(
        address _l2Token,
        string memory _name,
        string memory _symbol
    ) external {
        require(_l2Token != address(0), "Must provide L2 token address");

        L1StandardERC20 l1Token = new L1StandardERC20(
            Lib_PredeployAddresses.L1_STANDARD_BRIDGE,
            _l2Token,
            _name,
            _symbol
        );

        emit StandardL1TokenCreated(_l2Token, address(l1Token));
    }
}
