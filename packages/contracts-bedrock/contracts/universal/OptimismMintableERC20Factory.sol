// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Contract Imports */
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";
import { PredeployAddresses } from "../libraries/PredeployAddresses.sol";

/**
 * @custom:proxied
 * @custom:predeployed 0x4200000000000000000000000000000000000012
 * @title OptimismMintableERC20Factory
 * @notice OptimismMintableERC20Factory is a factory contract that generates OptimismMintableERC20
 *         contracts on the network it's deployed to. Simplifies the deployment process for users
 *         who may be less familiar with deploying smart contracts. Designed to be backwards
 *         compatible with the older StandardL2ERC20Factory contract.
 */
contract OptimismMintableERC20Factory {
    /**
     * @custom:legacy
     * @notice Emitted whenever a new OptimismMintableERC20 is created. Legacy version of the newer
     *         OptimismMintableERC20Created event. We recommend relying on that event instead.
     *
     * @param remoteToken Address of the token on the remote chain.
     * @param localToken  Address of the created token on the local chain.
     */
    event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);

    /**
     * @notice Emitted whenever a new OptimismMintableERC20 is created.
     *
     * @param localToken  Address of the created token on the local chain.
     * @param remoteToken Address of the corresponding token on the remote chain.
     * @param deployer    Address of the account that deployed the token.
     */
    event OptimismMintableERC20Created(
        address indexed localToken,
        address indexed remoteToken,
        address deployer
    );

    /**
     * @notice Address of the StandardBridge on this chain.
     */
    address public immutable bridge;

    /**
     * @param _bridge Address of the StandardBridge on this chain.
     */
    constructor(address _bridge) {
        bridge = _bridge;
    }

    /**
     * @custom:legacy
     * @notice Creates an instance of the OptimismMintableERC20 contract. Legacy version of the
     *         newer createOptimismMintableERC20 function, which has a more intuitive name.
     *
     * @param _remoteToken Address of the token on the remote chain.
     * @param _name        ERC20 name.
     * @param _symbol      ERC20 symbol.
     *
     * @return Address of the newly created token.
     */
    function createStandardL2Token(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) external returns (address) {
        return createOptimismMintableERC20(_remoteToken, _name, _symbol);
    }

    /**
     * @notice Creates an instance of the OptimismMintableERC20 contract.
     *
     * @param _remoteToken Address of the token on the remote chain.
     * @param _name        ERC20 name.
     * @param _symbol      ERC20 symbol.
     *
     * @return Address of the newly created token.
     */
    function createOptimismMintableERC20(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) public returns (address) {
        require(
            _remoteToken != address(0),
            "OptimismMintableERC20Factory: must provide remote token address"
        );

        require(
            bridge != address(0),
            "OptimismMintableERC20Factory: must initialize contract first"
        );

        OptimismMintableERC20 localToken = new OptimismMintableERC20(
            bridge,
            _remoteToken,
            _name,
            _symbol
        );

        // Emit the old event too for legacy support.
        emit StandardL2TokenCreated(_remoteToken, address(localToken));
        emit OptimismMintableERC20Created(_remoteToken, address(localToken), msg.sender);

        return address(localToken);
    }
}
