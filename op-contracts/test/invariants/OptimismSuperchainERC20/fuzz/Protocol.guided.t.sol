// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { MockL2ToL2CrossDomainMessenger } from "../helpers/MockL2ToL2CrossDomainMessenger.t.sol";
import { OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20.sol";
import { ProtocolHandler } from "../handlers/Protocol.t.sol";
import { EnumerableMap } from "@openzeppelin/contracts/utils/structs/EnumerableMap.sol";
import { CompatibleAssert } from "../helpers/CompatibleAssert.t.sol";

contract ProtocolGuided is ProtocolHandler, CompatibleAssert {
    using EnumerableMap for EnumerableMap.Bytes32ToUintMap;
    /// @notice deploy a new supertoken with deploy salt determined by params, to the given (of course mocked) chainId
    /// @custom:property-id 14
    /// @custom:property supertoken total supply starts at zero

    function fuzz_deployNewSupertoken(
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
        compatibleAssert(supertoken.totalSupply() == 0);
    }

    /// @custom:property-id 6
    /// @custom:property calls to sendERC20 succeed as long as caller has enough balance
    /// @custom:property-id 22
    /// @custom:property sendERC20 decreases sender balance in source chain and increases receiver balance in
    /// destination chain exactly by the input amount
    /// @custom:property-id 23
    /// @custom:property sendERC20 decreases total supply in source chain and increases it in destination chain exactly
    /// by the input amount
    function fuzz_bridgeSupertokenAtomic(
        uint256 fromIndex,
        uint256 recipientIndex,
        uint256 destinationChainId,
        uint256 amount
    )
        public
        withActor(msg.sender)
    {
        destinationChainId = bound(destinationChainId, 0, MAX_CHAINS - 1);
        fromIndex = bound(fromIndex, 0, allSuperTokens.length - 1);
        address recipient = getActorByRawIndex(recipientIndex);
        OptimismSuperchainERC20 sourceToken = OptimismSuperchainERC20(allSuperTokens[fromIndex]);
        OptimismSuperchainERC20 destinationToken =
            MESSENGER.crossChainMessageReceiver(address(sourceToken), destinationChainId);
        uint256 sourceBalanceBefore = sourceToken.balanceOf(currentActor());
        uint256 sourceSupplyBefore = sourceToken.totalSupply();
        uint256 destinationBalanceBefore = destinationToken.balanceOf(recipient);
        uint256 destinationSupplyBefore = destinationToken.totalSupply();

        MESSENGER.setAtomic(true);
        vm.prank(currentActor());
        try sourceToken.sendERC20(recipient, amount, destinationChainId) {
            MESSENGER.setAtomic(false);
            uint256 sourceBalanceAfter = sourceToken.balanceOf(currentActor());
            uint256 destinationBalanceAfter = destinationToken.balanceOf(recipient);
            // no free mint
            compatibleAssert(
                sourceBalanceBefore + destinationBalanceBefore == sourceBalanceAfter + destinationBalanceAfter
            );
            // 22
            compatibleAssert(sourceBalanceBefore - amount == sourceBalanceAfter);
            compatibleAssert(destinationBalanceBefore + amount == destinationBalanceAfter);
            uint256 sourceSupplyAfter = sourceToken.totalSupply();
            uint256 destinationSupplyAfter = destinationToken.totalSupply();
            // 23
            compatibleAssert(sourceSupplyBefore - amount == sourceSupplyAfter);
            compatibleAssert(destinationSupplyBefore + amount == destinationSupplyAfter);
        } catch {
            MESSENGER.setAtomic(false);
            // 6
            compatibleAssert(address(destinationToken) == address(sourceToken) || sourceBalanceBefore < amount);
        }
    }

    /// @custom:property-id 6
    /// @custom:property calls to sendERC20 succeed as long as caller has enough balance
    /// @custom:property-id 26
    /// @custom:property sendERC20 decreases sender balance in source chain exactly by the input amount
    /// @custom:property-id 10
    /// @custom:property sendERC20 decreases total supply in source chain exactly by the input amount
    function fuzz_sendERC20(
        uint256 fromIndex,
        uint256 recipientIndex,
        uint256 destinationChainId,
        uint256 amount
    )
        public
        withActor(msg.sender)
    {
        destinationChainId = bound(destinationChainId, 0, MAX_CHAINS - 1);
        fromIndex = bound(fromIndex, 0, allSuperTokens.length - 1);
        address recipient = getActorByRawIndex(recipientIndex);
        OptimismSuperchainERC20 sourceToken = OptimismSuperchainERC20(allSuperTokens[fromIndex]);
        OptimismSuperchainERC20 destinationToken =
            MESSENGER.crossChainMessageReceiver(address(sourceToken), destinationChainId);
        bytes32 deploySalt = MESSENGER.superTokenInitDeploySalts(address(sourceToken));
        uint256 sourceBalanceBefore = sourceToken.balanceOf(currentActor());
        uint256 sourceSupplyBefore = sourceToken.totalSupply();

        vm.prank(currentActor());
        try sourceToken.sendERC20(recipient, amount, destinationChainId) {
            (, uint256 currentlyInTransit) = ghost_tokensInTransit.tryGet(deploySalt);
            ghost_tokensInTransit.set(deploySalt, currentlyInTransit + amount);
            // 26
            uint256 sourceBalanceAfter = sourceToken.balanceOf(currentActor());
            compatibleAssert(sourceBalanceBefore - amount == sourceBalanceAfter);
            // 10
            uint256 sourceSupplyAfter = sourceToken.totalSupply();
            compatibleAssert(sourceSupplyBefore - amount == sourceSupplyAfter);
        } catch {
            // 6
            compatibleAssert(address(destinationToken) == address(sourceToken) || sourceBalanceBefore < amount);
        }
    }

    /// @custom:property-id 11
    /// @custom:property relayERC20 increases the token's totalSupply in the destination chain exactly by the input
    /// amount
    /// @custom:property-id 27
    /// @custom:property relayERC20 increases sender's balance in the destination chain exactly by the input amount
    /// @custom:property-id 7
    /// @custom:property calls to relayERC20 always succeed as long as the cross-domain caller is valid
    function fuzz_relayERC20(uint256 messageIndex) external {
        MockL2ToL2CrossDomainMessenger.CrossChainMessage memory messageToRelay = MESSENGER.messageQueue(messageIndex);
        OptimismSuperchainERC20 destinationToken = OptimismSuperchainERC20(messageToRelay.crossDomainMessageSender);
        uint256 destinationSupplyBefore = destinationToken.totalSupply();
        uint256 destinationBalanceBefore = destinationToken.balanceOf(messageToRelay.recipient);

        try MESSENGER.relayMessageFromQueue(messageIndex) {
            bytes32 deploySalt = MESSENGER.superTokenInitDeploySalts(address(destinationToken));
            (bool success, uint256 currentlyInTransit) = ghost_tokensInTransit.tryGet(deploySalt);
            // if sendERC20 didnt intialize this, then test suite is broken
            compatibleAssert(success);
            ghost_tokensInTransit.set(deploySalt, currentlyInTransit - messageToRelay.amount);
            // 11
            compatibleAssert(destinationSupplyBefore + messageToRelay.amount == destinationToken.totalSupply());
            // 27
            compatibleAssert(
                destinationBalanceBefore + messageToRelay.amount == destinationToken.balanceOf(messageToRelay.recipient)
            );
        } catch {
            // 7
            compatibleAssert(false);
        }
    }

    /// @custom:property-id 8
    /// @custom:property calls to sendERC20 with a value of zero dont modify accounting
    // @notice is a subset of fuzz_sendERC20, so we'll just call it
    // instead of re-implementing it. Keeping the function for visibility of the property.
    function fuzz_sendZeroDoesNotModifyAccounting(
        uint256 fromIndex,
        uint256 recipientIndex,
        uint256 destinationChainId
    )
        external
    {
        fuzz_sendERC20(fromIndex, recipientIndex, destinationChainId, 0);
    }

    /// @custom:property-id 9
    /// @custom:property calls to relayERC20 with a value of zero dont modify accounting
    /// @custom:property-id 7
    /// @custom:property calls to relayERC20 always succeed as long as the cross-domain caller is valid
    /// @notice cant call fuzz_RelayERC20 internally since that pops a
    /// random message, which we cannot guarantee has a value of zero
    function fuzz_relayZeroDoesNotModifyAccounting(
        uint256 fromIndex,
        uint256 recipientIndex
    )
        external
        withActor(msg.sender)
    {
        fromIndex = bound(fromIndex, 0, allSuperTokens.length - 1);
        address recipient = getActorByRawIndex(recipientIndex);
        OptimismSuperchainERC20 token = OptimismSuperchainERC20(allSuperTokens[fromIndex]);
        uint256 balanceSenderBefore = token.balanceOf(currentActor());
        uint256 balanceRecipientBefore = token.balanceOf(recipient);
        uint256 supplyBefore = token.totalSupply();

        MESSENGER.setCrossDomainMessageSender(address(token));
        vm.prank(address(MESSENGER));
        try token.relayERC20(currentActor(), recipient, 0) {
            MESSENGER.setCrossDomainMessageSender(address(0));
        } catch {
            // should not revert because of 7, and if it *does* revert, I want the test suite
            // to discard the sequence instead of potentially getting another
            // error due to the crossDomainMessageSender being manually set
            compatibleAssert(false);
        }
        uint256 balanceSenderAfter = token.balanceOf(currentActor());
        uint256 balanceRecipeintAfter = token.balanceOf(recipient);
        uint256 supplyAfter = token.totalSupply();
        compatibleAssert(balanceSenderBefore == balanceSenderAfter);
        compatibleAssert(balanceRecipientBefore == balanceRecipeintAfter);
        compatibleAssert(supplyBefore == supplyAfter);
    }
}
