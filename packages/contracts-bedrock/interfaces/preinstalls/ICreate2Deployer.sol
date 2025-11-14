// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

interface ICreate2Deployer {
    /**
     * @dev Deploys a contract using `CREATE2`. The address where the
     * contract will be deployed can be known in advance via {computeAddress}.
     *
     * The bytecode for a contract can be obtained from Solidity with
     * `type(contractName).creationCode`.
     *
     * Requirements:
     * - `bytecode` must not be empty.
     * - `salt` must have not been used for `bytecode` already.
     * - the factory must have a balance of at least `value`.
     * - if `value` is non-zero, `bytecode` must have a `payable` constructor.
     */
    function deploy(uint256 value, bytes32 salt, bytes memory code) external;
    /**
     * @dev Deployment of the {ERC1820Implementer}.
     * Further information: https://eips.ethereum.org/EIPS/eip-1820
     */
    function deployERC1820Implementer(uint256 value, bytes32 salt) external;
    /**
     * @dev Returns the address where a contract will be stored if deployed via {deploy}.
     * Any change in the `bytecodeHash` or `salt` will result in a new destination address.
     */
    function computeAddress(bytes32 salt, bytes32 codeHash) external view returns (address);
    /**
     * @dev Returns the address where a contract will be stored if deployed via {deploy} from a
     * contract located at `deployer`. If `deployer` is this contract's address, returns the
     * same value as {computeAddress}.
     */
    function computeAddressWithDeployer(
        bytes32 salt,
        bytes32 codeHash,
        address deployer
    )
        external
        pure
        returns (address);

    receive() external payable;
}
