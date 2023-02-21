pragma solidity 0.8.15;

import { L2OutputOracle_Initializer } from "../CommonTest.t.sol";

contract L2OutputOracle_MonotonicBlockNumIncrease_Invariant is L2OutputOracle_Initializer {
    function setUp() public override {
        super.setUp();

        // Set the target contract to the oracle proxy
        targetContract(address(oracle));
        // Set the target sender to the proposer
        targetSender(address(proposer));
        // Set the target selector for `proposeL2Output`
        // `proposeL2Output` is the only function we care about, as it is the only function
        // that can modify the `l2Outputs` array in the oracle.
        bytes4[] memory selectors = new bytes4[](1);
        selectors[0] = oracle.proposeL2Output.selector;
        FuzzSelector memory selector = FuzzSelector({
            addr: address(oracle),
            selectors: selectors
        });
        targetSelector(selector);
    }

    /**
     * @custom:invariant The block number of the output root proposals should monotonically
     * increase.
     *
     * When a new output is submitted, it should never be allowed to correspond to a block
     * number that is less than the current output.
     */
    function invariant_monotonicBlockNumIncrease() external {
        // Assert that the block number of proposals must monotonically increase.
        assertTrue(oracle.nextBlockNumber() >= oracle.latestBlockNumber());
    }
}
