// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Import this here to make it available just by importing this file
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";


/**
 * @title IRemoteToken
 * @notice This interface is available on the OptimismMintableERC20 contract. We declare it as a
 *         separate interface so that it can be used in custom implementations of
 *         OptimismMintableERC20.
 */
interface IHasRemoteToken {
    function remoteToken() external;

    function mint(address _to, uint256 _amount) external;

    function burn(address _from, uint256 _amount) external;
}

/**
 * @custom:legacy
 * @title IHasL1Token
 * @notice This interface was available on the legacy L2StandardERC20 contract. It remains available
 *         on the OptimismMintableERC20 contract for backwards compatibility.
 */
interface IHasL1Token {
    function l1Token() external;

    function mint(address _to, uint256 _amount) external;

    function burn(address _from, uint256 _amount) external;
}
