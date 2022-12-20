pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { SystemConfig } from "../../L1/SystemConfig.sol";

contract SystemConfig_Invariant is Test {
    SystemConfig config;

    function setUp() public {
        config = new SystemConfig({
            _owner: address(this),
            _overhead: 2100,
            _scalar: 1000000,
            _batcherHash: bytes32(hex"abcd"),
            _gasLimit: 8_000_000,
            _unsafeBlockSigner: address(1)
        });
    }

    /**
     * INVARIANT: The gas limit of the `SystemConfig` contract can never be lower
     * than the hard-coded lower bound.
     */
    function invariant_gasLimitLowerBound() external {
        assertTrue(config.gasLimit() >= config.MINIMUM_GAS_LIMIT());
    }
}
