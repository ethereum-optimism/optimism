import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { Types } from "../libraries/Types.sol";

contract EchidnaL2OutputOracle is L2OutputOracle {
    constructor(
        uint256 _submissionInterval,
        bytes32 _genesisL2Output,
        uint256 _historicalTotalBlocks,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        uint256 _l2BlockTime
    ) L2OutputOracle(
        _submissionInterval,
        _genesisL2Output,
        _historicalTotalBlocks,
        _startingBlockNumber,
        _startingTimestamp,
        _l2BlockTime,
        address(0xAbBa),
        address(0xACDC)
    ) {
        // standard initialization migrates both proposer and owner from the
        // deployer to new addresses and prevents them from being the same
        // address. however, we need the Fuzz[...] contract, the deployer in
        // this case, to be able to propose to the oracle, so we extend it
        // just to be able to force the proposer role back on to the testing
        // contract (we don't care what addresses are passed to the constructor
        //  so we borrow from CommonTest.t.sol again
        proposer = msg.sender;
    }
}

contract FuzzOptimismPortalWithdrawals{
    // since portal.finalizedWithdrawals is declared `external`, we can't use helper
    // functions to check state before/after withdrawals, so we split these tests
    // into a separate contract that only interacts externally
    OptimismPortal portal;
    L2OutputOracle oracle;

    Types.WithdrawalTransaction cachedTx;
    uint256 cachedL2BlockNumber;
    Types.OutputRootProof cachedOutputRootProof;
    bytes cachedWithdrawalProof;

    uint256 offset;

    bool failedFinalizeEarly;
    bool failedMismatchedOutputRoot;

    constructor() {
        // seeding with the values from CommonTest.t.sol
        oracle = new EchidnaL2OutputOracle(1800, keccak256(abi.encode(0)), 199, 200, 1000, 2);
        portal = new OptimismPortal(oracle, 7 days);
    }

    /**
     * @notice Submits a proposal to the L2OutputOracle. Currently completely unstructured
     */
    function proposeL2Output(
        bytes32 _outputRoot,
        bytes32 _l2BlockNumber,
        bytes32 _l1Blockhash,
        uint256 _l1BlockNumber
    ) public {
        // TODO: provide more structure for this data
        oracle.proposeL2Output(_outputRoot, oracle.nextBlockNumber(), _l1Blockhash, _l1BlockNumber);
    }

    /**
     * @notice Generates a withdrawal transaction and (TODO) performs the necessary state changes
     *         to ensure it will validate
     */
    function finalizeValidWithdrawal(
        Types.WithdrawalTransaction calldata _tx,
        uint256 _l2BlockNumber,
        Types.OutputRootProof calldata _outputRootProof,
        bytes calldata _withdrawalProof
    ) public {
        // TODO: craft a valid call to oracle.proposeL2Output for this generated withdrawal

        portal.finalizeWithdrawalTransaction(_tx, _l2BlockNumber, _outputRootProof, _withdrawalProof);

        // if we haven't reverted, we have a valid withdrawal transaction.
        // save it so we can test for replay protection
        cachedTx = _tx;
        cachedL2BlockNumber = _l2BlockNumber;
        cachedOutputRootProof = _outputRootProof;
        cachedWithdrawalProof = _withdrawalProof;
    }

    /**
     * @notice Generates a withdrawal transaction and (TODO) performs the necessary state changes
     *         such that it is an otherwise valid transaction, but the finalization period has not
     *         yet concluded.
     */
    function finalizeEarlyWithdrawal(
        Types.WithdrawalTransaction calldata _tx,
        uint256 _l2BlockNumber,
        Types.OutputRootProof calldata _outputRootProof,
        bytes calldata _withdrawalProof
    ) public {
        // TODO: craft a valid call to oracle.proposeL2Output, still within the finalization period

        portal.finalizeWithdrawalTransaction(_tx, _l2BlockNumber, _outputRootProof, _withdrawalProof);

        // Execution should have reverted prior to this since we're still within the finalization window
        failedFinalizeEarly = true;
    }

    /**
     * @notice Helper function so we can access some randomness in the echidna_* tests
     */
    function setOffset(uint256 _offset) public {
        offset = _offset;
    }

    function echidna_never_finalize_early() public view returns (bool) {
        return !failedFinalizeEarly;
    }

    function echidna_never_finalize_twice() public returns (bool) {
        if (cachedL2BlockNumber != 0) {
            portal.finalizeWithdrawalTransaction(cachedTx, cachedL2BlockNumber, cachedOutputRootProof, cachedWithdrawalProof);
            return false;
        }
        return true;
    }

    function echidna_never_finalize_no_oracle_data() public returns (bool) {
        if (cachedL2BlockNumber != 0) {
            portal.finalizeWithdrawalTransaction(cachedTx, (oracle.nextBlockNumber() + (offset % 5000)), cachedOutputRootProof, cachedWithdrawalProof);
            return false;
        }
        return true;
    }
}