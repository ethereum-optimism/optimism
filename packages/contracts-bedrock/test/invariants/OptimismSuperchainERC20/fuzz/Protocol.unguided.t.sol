// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { ProtocolHandler } from "../handlers/Protocol.t.sol";
import { EnumerableMap } from "@openzeppelin/contracts/utils/structs/EnumerableMap.sol";
import { OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20.sol";
import { CompatibleAssert } from "../helpers/CompatibleAssert.t.sol";

// TODO: add fuzz_sendERC20 when we implement non-atomic bridging
contract ProtocolUnguided is ProtocolHandler, CompatibleAssert {
    using EnumerableMap for EnumerableMap.Bytes32ToUintMap;

    /// @custom:property-id 7
    /// @custom:property calls to relayERC20 always succeed as long as the cross-domain caller is valid
    /// @notice this ensures actors cant simply call relayERC20 and get tokens, no matter the system state
    /// but there's still some possible work on how hard we can bork the system state with handlers calling
    /// the L2ToL2CrossDomainMessenger or bridge directly (pending on non-atomic bridging)
    function fuzz_relayERC20(
        uint256 tokenIndex,
        address sender,
        address crossDomainMessageSender,
        address recipient,
        uint256 amount
    )
        external
    {
        MESSENGER.setCrossDomainMessageSender(crossDomainMessageSender);
        address token = allSuperTokens[bound(tokenIndex, 0, allSuperTokens.length)];
        vm.prank(sender);
        try OptimismSuperchainERC20(token).relayERC20(sender, recipient, amount) {
            MESSENGER.setCrossDomainMessageSender(address(0));
            compatibleAssert(sender == address(MESSENGER));
            compatibleAssert(crossDomainMessageSender == token);
            // this increases the supply across chains without a call to
            // `mint` by the MESSENGER, so it kind of breaks an invariant, but
            // let's walk around that:
            bytes32 salt = MESSENGER.superTokenInitDeploySalts(token);
            (, uint256 currentValue) = ghost_totalSupplyAcrossChains.tryGet(salt);
            ghost_totalSupplyAcrossChains.set(salt, currentValue + amount);
        } catch {
            compatibleAssert(sender != address(MESSENGER) || crossDomainMessageSender != token);
            MESSENGER.setCrossDomainMessageSender(address(0));
        }
    }

    /// @custom:property-id 6
    /// @custom:property calls to sendERC20 succeed as long as caller has enough balance
    /// @custom:property-id 26
    /// @custom:property sendERC20 decreases sender balance in source chain exactly by the input amount
    /// @custom:property-id 10
    /// @custom:property sendERC20 decreases total supply in source chain exactly by the input amount
    function fuzz_sendERC20(
        address sender,
        address recipient,
        uint256 fromIndex,
        uint256 destinationChainId,
        uint256 amount
    )
        public
    {
        destinationChainId = bound(destinationChainId, 0, MAX_CHAINS - 1);
        OptimismSuperchainERC20 sourceToken = OptimismSuperchainERC20(allSuperTokens[fromIndex]);
        OptimismSuperchainERC20 destinationToken =
            MESSENGER.crossChainMessageReceiver(address(sourceToken), destinationChainId);
        bytes32 deploySalt = MESSENGER.superTokenInitDeploySalts(address(sourceToken));
        uint256 sourceBalanceBefore = sourceToken.balanceOf(sender);
        uint256 sourceSupplyBefore = sourceToken.totalSupply();

        vm.prank(sender);
        try sourceToken.sendERC20(recipient, amount, destinationChainId) {
            (, uint256 currentlyInTransit) = ghost_tokensInTransit.tryGet(deploySalt);
            ghost_tokensInTransit.set(deploySalt, currentlyInTransit + amount);
            // 26
            uint256 sourceBalanceAfter = sourceToken.balanceOf(sender);
            compatibleAssert(sourceBalanceBefore - amount == sourceBalanceAfter);
            // 10
            uint256 sourceSupplyAfter = sourceToken.totalSupply();
            compatibleAssert(sourceSupplyBefore - amount == sourceSupplyAfter);
        } catch {
            // 6
            compatibleAssert(address(destinationToken) == address(sourceToken) || sourceBalanceBefore < amount);
        }
    }

    /// @custom:property-id 12
    /// @custom:property supertoken total supply only increases on calls to mint() by the L2toL2StandardBridge
    function fuzz_mint(uint256 tokenIndex, address to, address sender, uint256 amount) external {
        address token = allSuperTokens[bound(tokenIndex, 0, allSuperTokens.length)];
        bytes32 salt = MESSENGER.superTokenInitDeploySalts(token);
        amount = bound(amount, 0, type(uint256).max - OptimismSuperchainERC20(token).totalSupply());
        vm.prank(sender);
        try OptimismSuperchainERC20(token).mint(to, amount) {
            compatibleAssert(sender == BRIDGE);
            (, uint256 currentValue) = ghost_totalSupplyAcrossChains.tryGet(salt);
            ghost_totalSupplyAcrossChains.set(salt, currentValue + amount);
        } catch {
            compatibleAssert(sender != BRIDGE || to == address(0));
        }
    }

    /// @custom:property-id 13
    /// @custom:property supertoken total supply only increases on calls to mint() by the L2toL2StandardBridge
    function fuzz_burn(uint256 tokenIndex, address from, address sender, uint256 amount) external {
        address token = allSuperTokens[bound(tokenIndex, 0, allSuperTokens.length)];
        bytes32 salt = MESSENGER.superTokenInitDeploySalts(token);
        uint256 senderBalance = OptimismSuperchainERC20(token).balanceOf(sender);
        vm.prank(sender);
        try OptimismSuperchainERC20(token).burn(from, amount) {
            compatibleAssert(sender == BRIDGE);
            (, uint256 currentValue) = ghost_totalSupplyAcrossChains.tryGet(salt);
            ghost_totalSupplyAcrossChains.set(salt, currentValue - amount);
        } catch {
            compatibleAssert(sender != BRIDGE || senderBalance < amount);
        }
    }

    /// @custom:property-id 25
    /// @custom:property supertokens can't be reinitialized
    function fuzz_initialize(
        address sender,
        uint256 tokenIndex,
        address remoteToken,
        string memory name,
        string memory symbol,
        uint8 decimals
    )
        external
    {
        vm.prank(sender);
        // revert is possible in bound, but is not part of the external call
        try OptimismSuperchainERC20(allSuperTokens[bound(tokenIndex, 0, allSuperTokens.length)]).initialize(
            remoteToken, name, symbol, decimals
        ) {
            compatibleAssert(false);
        } catch { }
    }
}
