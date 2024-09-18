// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test, StdUtils, Vm } from "forge-std/Test.sol";
import { StdInvariant } from "forge-std/StdInvariant.sol";
import { StdAssertions } from "forge-std/StdAssertions.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";
import { ProtocolGuided } from "./fuzz/Protocol.guided.t.sol";
import { ProtocolUnguided } from "./fuzz/Protocol.unguided.t.sol";
import { HandlerGetters } from "./helpers/HandlerGetters.t.sol";
import { MockL2ToL2CrossDomainMessenger } from "./helpers/MockL2ToL2CrossDomainMessenger.t.sol";

contract OptimismSuperchainERC20Handler is HandlerGetters, ProtocolGuided, ProtocolUnguided { }

contract OptimismSuperchainERC20Properties is Test {
    OptimismSuperchainERC20Handler internal handler;
    MockL2ToL2CrossDomainMessenger internal constant MESSENGER =
        MockL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

    function setUp() public {
        handler = new OptimismSuperchainERC20Handler();
        targetContract(address(handler));
    }

    // TODO: will need rework after
    //   - `convert`
    /// @custom:invariant sum of supertoken total supply across all chains is always <= to convert(legacy, super)-
    /// convert(super, legacy)
    function invariant_totalSupplyAcrossChainsEqualsMintsMinusFundsInTransit() external view {
        // iterate over unique deploy salts aka supertokens that are supposed to be compatible with each other
        for (uint256 deploySaltIndex = 0; deploySaltIndex < handler.deploySaltsLength(); deploySaltIndex++) {
            uint256 totalSupply = 0;
            (bytes32 currentSalt, uint256 trackedSupply) = handler.totalSupplyAcrossChainsAtIndex(deploySaltIndex);
            uint256 fundsInTransit = handler.tokensInTransitForDeploySalt(currentSalt);
            // and then over all the (mocked) chain ids where that supertoken could be deployed
            for (uint256 validChainId = 0; validChainId < handler.MAX_CHAINS(); validChainId++) {
                address supertoken = MESSENGER.superTokenAddresses(validChainId, currentSalt);
                if (supertoken != address(0)) {
                    totalSupply += OptimismSuperchainERC20(supertoken).totalSupply();
                }
            }
            assertEq(trackedSupply, totalSupply + fundsInTransit);
        }
    }

    // TODO: will need rework after
    //   - `convert`
    /// @custom:invariant sum of supertoken total supply across all chains is equal to convert(legacy, super)-
    /// convert(super, legacy) when all when all cross-chain messages are processed
    function invariant_totalSupplyAcrossChainsEqualsMintsWhenQueueIsEmpty() external view {
        if (MESSENGER.messageQueueLength() != 0) {
            return;
        }
        // iterate over unique deploy salts aka supertokens that are supposed to be compatible with each other
        for (uint256 deploySaltIndex = 0; deploySaltIndex < handler.deploySaltsLength(); deploySaltIndex++) {
            uint256 totalSupply = 0;
            (bytes32 currentSalt, uint256 trackedSupply) = handler.totalSupplyAcrossChainsAtIndex(deploySaltIndex);
            // and then over all the (mocked) chain ids where that supertoken could be deployed
            for (uint256 validChainId = 0; validChainId < handler.MAX_CHAINS(); validChainId++) {
                address supertoken = MESSENGER.superTokenAddresses(validChainId, currentSalt);
                if (supertoken != address(0)) {
                    totalSupply += OptimismSuperchainERC20(supertoken).totalSupply();
                }
            }
            assertEq(trackedSupply, totalSupply);
        }
    }

    /// @custom:invariant many other assertion mode invariants are also defined  under
    /// `test/invariants/OptimismSuperchainERC20/fuzz/` .
    ///
    ///     since setting`fail_on_revert=false` also ignores StdAssertion failures, this invariant explicitly asks the
    ///     handler for assertion test failures
    function invariant_handlerAssertions() external view {
        assertFalse(handler.failed());
    }
}
