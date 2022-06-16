// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Contract Imports */
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";
import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";

/**
 * @title OptimismMintableTokenFactory
 * @dev Factory contract for creating standard remote token representations of
 * local ERC20s. This can be used to bridge native L1 ERC20s to L2 or native L2
 * ERC20s to L1. The tokens created through this factory are meant to operate
 * with the StandardBridge contract for deposits/withdrawals.
 * This contract is a predeploy on L2 at 0x4200000000000000000000000000000000000012
 * TODO: deploy to a deterministic address on L1 networks?
 * TODO: should this be extended for L1/L2 with hardcoded values in
 * the base contract's initialize?
 */
contract OptimismMintableTokenFactory {
    event StandardL2TokenCreated(address indexed _remoteToken, address indexed _localToken);
    event OptimismMintableTokenCreated(
        address indexed _localToken,
        address indexed _remoteToken,
        address _deployer
    );

    address public bridge;

    /**
     * @dev Initialize the factory
     * On L2 _bridge should be Lib_PredeployAddresses.L2_STANDARD_BRIDGE,
     * On L1 _bridge should be the L1StandardBridge
     */
    function initialize(address _bridge) public {
        require(bridge == address(0), "Already initialized.");
        bridge = _bridge;
    }

    /**
     * @dev Creates an instance of the standard ERC20 token on L2.
     * @param _remoteToken Address of the corresponding L1 token.
     * @param _name ERC20 name.
     * @param _symbol ERC20 symbol.
     * @return Address of the new token.
     */
    function createStandardL2Token(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) external returns (address) {
        require(_remoteToken != address(0), "Must provide L1 token address");
        require(bridge != address(0), "Must initialize first");

        OptimismMintableERC20 localToken = new OptimismMintableERC20(
            bridge,
            _remoteToken,
            _name,
            _symbol
        );

        // Legacy Purposes
        emit StandardL2TokenCreated(_remoteToken, address(localToken));
        emit OptimismMintableTokenCreated(_remoteToken, address(localToken), msg.sender);

        return address(localToken);
    }
}
