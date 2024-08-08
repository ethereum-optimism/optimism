// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { ProtocolHandler } from "./handlers/Protocol.handler.t.sol";
import { OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20.sol";
import { EnumerableMap } from "@openzeppelin/contracts/utils/structs/EnumerableMap.sol";

contract ProtocolProperties is ProtocolHandler {
    using EnumerableMap for EnumerableMap.Bytes32ToUintMap;

    // TODO: will need rework after
    //   - non-atomic bridge
    //   - `convert`
    /// @custom:property-id 24
    /// @custom:property sum of supertoken total supply across all chains is always equal to convert(legacy, super)-
    /// convert(super, legacy)
    function property_totalSupplyAcrossChainsEqualsMints() external view {
        // iterate over unique deploy salts aka supertokens that are supposed to be compatible with each other
        for (uint256 deploySaltIndex = 0; deploySaltIndex < ghost_totalSupplyAcrossChains.length(); deploySaltIndex++) {
            uint256 totalSupply = 0;
            (bytes32 currentSalt, uint256 trackedSupply) = ghost_totalSupplyAcrossChains.at(deploySaltIndex);
            // and then over all the (mocked) chain ids where that supertoken could be deployed
            for (uint256 validChainId = 0; validChainId < MAX_CHAINS; validChainId++) {
                address supertoken = MESSENGER.superTokenAddresses(validChainId, currentSalt);
                if (supertoken != address(0)) {
                    totalSupply += OptimismSuperchainERC20(supertoken).totalSupply();
                }
            }
            assert(trackedSupply == totalSupply);
        }
    }

    /// @notice deploy a new supertoken with deploy salt determined by params, to the given (of course mocked) chainId
    /// @custom:property-id 14
    /// @custom:property supertoken total supply starts at zero
    function property_DeployNewSupertoken(
        TokenDeployParams memory params,
        uint256 chainId
    )
        external
        validateTokenDeployParams(params)
    {
        chainId = bound(chainId, 0, MAX_CHAINS - 1);
        OptimismSuperchainERC20 supertoken = _deploySupertoken(
            remoteTokens[params.remoteTokenIndex],
            WORDS[params.nameIndex],
            WORDS[params.symbolIndex],
            DECIMALS[params.decimalsIndex],
            chainId
        );
        // 14
        assert(supertoken.totalSupply() == 0);
    }

    /// @custom:property-id 22
    /// @custom:property sendERC20 decreases sender balance in source chain and increases receiver balance in
    /// destination chain exactly by the input amount
    /// @custom:property-id 23
    /// @custom:property sendERC20 decreases total supply in source chain and increases it in destination chain exactly
    /// by the input amount
    function property_SelfBridgeSupertoken(uint256 fromIndex, uint256 destinationChainId, uint256 amount) external {
        destinationChainId = bound(destinationChainId, 0, MAX_CHAINS - 1);
        fromIndex = bound(fromIndex, 0, allSuperTokens.length - 1);
        OptimismSuperchainERC20 sourceToken = OptimismSuperchainERC20(allSuperTokens[fromIndex]);
        OptimismSuperchainERC20 destinationToken =
            MESSENGER.crossChainMessageReceiver(address(sourceToken), destinationChainId);
        // TODO: when implementing non-atomic bridging, allow for the token to
        // not yet be deployed and funds be recovered afterwards.
        require(address(destinationToken) != address(0));
        uint256 sourceBalanceBefore = sourceToken.balanceOf(msg.sender);
        uint256 sourceSupplyBefore = sourceToken.totalSupply();
        uint256 destinationBalanceBefore = destinationToken.balanceOf(msg.sender);
        uint256 destinationSupplyBefore = destinationToken.totalSupply();

        vm.prank(msg.sender);
        try sourceToken.sendERC20(msg.sender, amount, destinationChainId) {
            uint256 sourceBalanceAfter = sourceToken.balanceOf(msg.sender);
            uint256 destinationBalanceAfter = destinationToken.balanceOf(msg.sender);
            // no free mint
            assert(sourceBalanceBefore + destinationBalanceBefore == sourceBalanceAfter + destinationBalanceAfter);
            // 22
            assert(sourceBalanceBefore - amount == sourceBalanceAfter);
            assert(destinationBalanceBefore + amount == destinationBalanceAfter);
            uint256 sourceSupplyAfter = sourceToken.totalSupply();
            uint256 destinationSupplyAfter = destinationToken.totalSupply();
            // 23
            assert(sourceSupplyBefore - amount == sourceSupplyAfter);
            assert(destinationSupplyBefore + amount == destinationSupplyAfter);
        } catch {
            assert(address(destinationToken) == address(sourceToken) || sourceBalanceBefore < amount);
        }
    }
}
