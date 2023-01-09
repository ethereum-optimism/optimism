// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Proxy } from "./Proxy.sol";

/**
 * @title HardforkOnlyProxy
 * @notice HardforkOnlyProxy is a special proxy contract that can ONLY be directly upgraded via a
 *         hardfork. The owner of this proxy contract is a special address with an unknown private
 *         key (0x1234000000000000000000000000000000004321). Transactions can only be triggered
 *         from the special owner address via a deposit transaction in the op-node, and no such
 *         logic currently exists inside the op-node. To create such a deposit transaction, the L2
 *         system would have to be upgraded via hardfork. All predeployed contracts exist behind
 *         a HardforkOnlyProxy to generally simplify the process of upgrading predeployed contracts
 *         via hardfork when necessary.
 */
contract HardforkOnlyProxy is Proxy {
    constructor() Proxy(0x1234000000000000000000000000000000004321) {}
}
