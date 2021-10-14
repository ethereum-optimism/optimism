// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Contract Imports */
import { L2StandardERC20 } from "../../standards/L2StandardERC20.sol";
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/**
 * @title L2StandardTokenFactory
 * @dev Factory contract for creating standard L2 token representations of L1 ERC20s
 * compatible with and working on the standard bridge.
 */
contract L2StandardTokenFactory {
    event StandardL2TokenCreated(address indexed _l1Token, address indexed _l2Token);

    /**
     * @dev Creates an instance of the standard ERC20 token on L2.
     * @param _l1Token Address of the corresponding L1 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     */
    function createStandardL2Token(
        address _l1Token,
        string memory _name,
        string memory _symbol
    ) external {
        require(_l1Token != address(0), "Must provide L1 token address");

        L2StandardERC20 l2Token = new L2StandardERC20(
            Lib_PredeployAddresses.L2_STANDARD_BRIDGE,
            _l1Token,
            _name,
            _symbol
        );

        emit StandardL2TokenCreated(_l1Token, address(l2Token));
    }
}
