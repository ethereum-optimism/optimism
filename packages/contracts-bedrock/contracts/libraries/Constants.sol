// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ResourceMetering } from "../L1/ResourceMetering.sol";

/**
 * @title Constants
 * @notice Constants is a library for storing constants. Simple! Don't put everything in here, just
 *         the stuff used in multiple contracts. Constants that only apply to a single contract
 *         should be defined in that contract instead.
 */
library Constants {
    /**
     * @notice Special address to be used as the tx origin for gas estimation calls in the
     *         OptimismPortal and CrossDomainMessenger calls. You only need to use this address if
     *         the minimum gas limit specified by the user is not actually enough to execute the
     *         given message and you're attempting to estimate the actual necessary gas limit. We
     *         use address(1) because it's the ecrecover precompile and therefore guaranteed to
     *         never have any code on any EVM chain.
     */
    address internal constant ESTIMATION_ADDRESS = address(1);

    /**
     * @notice Value used for the L2 sender storage slot in both the OptimismPortal and the
     *         CrossDomainMessenger contracts before an actual sender is set. This value is
     *         non-zero to reduce the gas cost of message passing transactions.
     */
    address internal constant DEFAULT_L2_SENDER = 0x000000000000000000000000000000000000dEaD;

    /**
     * @notice Returns the default values for the ResourceConfig. These are the recommended values
     *         for a production network.
     */
    function DEFAULT_RESOURCE_CONFIG()
        internal
        pure
        returns (ResourceMetering.ResourceConfig memory)
    {
        ResourceMetering.ResourceConfig memory config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            minimumBaseFee: 1 gwei,
            systemTxMaxGas: 1_000_000,
            maximumBaseFee: type(uint128).max
        });
        return config;
    }
}
