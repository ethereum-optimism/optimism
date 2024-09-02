// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { ITokenMock } from "@crytic/properties/contracts/ERC20/external/util/ITokenMock.sol";
import { EnumerableMap } from "@openzeppelin/contracts/utils/structs/EnumerableMap.sol";
import { ProtocolGuided } from "../fuzz/Protocol.guided.t.sol";
import { ProtocolUnguided } from "../fuzz/Protocol.unguided.t.sol";
import { CryticERC20ExternalBasicProperties } from
    "@crytic/properties/contracts/ERC20/external/properties/ERC20ExternalBasicProperties.sol";
import { OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20.sol";

contract ProtocolProperties is ProtocolGuided, ProtocolUnguided, CryticERC20ExternalBasicProperties {
    using EnumerableMap for EnumerableMap.Bytes32ToUintMap;

    /// @dev `token` is the token under test for the ToB properties. This is coupled
    /// to the ProtocolSetup constructor initializing at least one supertoken
    constructor() {
        token = ITokenMock(allSuperTokens[0]);
    }

    /// @dev not that much of a handler, since this only changes which
    /// supertoken the ToB assertions are performed against. Thankfully, they are
    /// implemented in a way that don't require tracking ghost variables or can
    /// break properties defined by us
    function handler_ToBTestOtherToken(uint256 index) external {
        token = ITokenMock(allSuperTokens[bound(index, 0, allSuperTokens.length - 1)]);
    }

    // TODO: will need rework after
    //   - `convert`
    /// @custom:property-id 19
    /// @custom:property sum of supertoken total supply across all chains is always <= to convert(legacy, super)-
    /// convert(super, legacy)
    function property_totalSupplyAcrossChainsEqualsMintsMinusFundsInTransit() external view {
        // iterate over unique deploy salts aka supertokens that are supposed to be compatible with each other
        for (uint256 deploySaltIndex ; deploySaltIndex < ghost_totalSupplyAcrossChains.length(); deploySaltIndex++) {
            uint256 totalSupply ;
            (bytes32 currentSalt, uint256 trackedSupply) = ghost_totalSupplyAcrossChains.at(deploySaltIndex);
            (, uint256 fundsInTransit) = ghost_tokensInTransit.tryGet(currentSalt);
            // and then over all the (mocked) chain ids where that supertoken could be deployed
            for (uint256 validChainId ; validChainId < MAX_CHAINS; validChainId++) {
                address supertoken = MESSENGER.superTokenAddresses(validChainId, currentSalt);
                if (supertoken != address(0)) {
                    totalSupply += OptimismSuperchainERC20(supertoken).totalSupply();
                }
            }
            assert(trackedSupply == totalSupply + fundsInTransit);
        }
    }

    // TODO: will need rework after
    //   - `convert`
    /// @custom:property-id 21
    /// @custom:property sum of supertoken total supply across all chains is equal to convert(legacy, super)-
    /// convert(super, legacy) when all when all cross-chain messages are processed
    function property_totalSupplyAcrossChainsEqualsMintsWhenQueueIsEmpty() external view {
        require(MESSENGER.messageQueueLength() == 0);
        // iterate over unique deploy salts aka supertokens that are supposed to be compatible with each other
        for (uint256 deploySaltIndex ; deploySaltIndex < ghost_totalSupplyAcrossChains.length(); deploySaltIndex++) {
            uint256 totalSupply ;
            (bytes32 currentSalt, uint256 trackedSupply) = ghost_totalSupplyAcrossChains.at(deploySaltIndex);
            // and then over all the (mocked) chain ids where that supertoken could be deployed
            for (uint256 validChainId ; validChainId < MAX_CHAINS; validChainId++) {
                address supertoken = MESSENGER.superTokenAddresses(validChainId, currentSalt);
                if (supertoken != address(0)) {
                    totalSupply += OptimismSuperchainERC20(supertoken).totalSupply();
                }
            }
            assert(trackedSupply == totalSupply);
        }
    }
}
