//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title L1Block
 * @dev This is an L2 predeploy contract that holds values from the L1
 * chain. It can only be updated by a special account that has no private
 * key managed by the L2 system. Transactions sent to this contract can
 * be thought of as "L2 system transactions".
 */
contract L1Block {
    /**
     * @notice Only the Depositor account may call setL1BlockValues().
     */
    error OnlyDepositor();

    /**
     * @notice The depositor account is a special account that sends
     * transactions to this contract.
     */
    address public constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;

    /**
     * @notice The latest L1 block number known by the L2 system
     */
    uint64 public number;

    /**
     * @notice The latest L1 timestamp known by the L2 system
     */
    uint64 public timestamp;

    /**
     * @notice The average L1 basefee
     */
    uint256 public basefee;

    /**
     * @notice The latest L1 blockhash
     */
    bytes32 public hash;

    /**
     * @notice The number of L2 blocks in the same epoch
     */
    uint64 public sequenceNumber;

    /**
     * @notice The number of samples to be taken into account
     * when computing the average L1 basefee
     */

    uint256 internal constant AVERAGE_BASEFEE_WINDOW = 256;

    /**
     * @notice The set of L1 basefees used to compute the
     * average L1 basefee
     */
    uint256[AVERAGE_BASEFEE_WINDOW] public basefees;

    /**
     * @notice Sets the L1 values
     * @param _number L1 blocknumber
     * @param _timestamp L1 timestamp
     * @param _basefee L1 basefee
     * @param _hash L1 blockhash
     * @param _sequenceNumber Number of L2 blocks since epoch start
     */
    function setL1BlockValues(
        uint64 _number,
        uint64 _timestamp,
        uint256 _basefee,
        bytes32 _hash,
        uint64 _sequenceNumber
    ) external {
        if (msg.sender != DEPOSITOR_ACCOUNT) {
            revert OnlyDepositor();
        }

        basefees[block.number % AVERAGE_BASEFEE_WINDOW] = _basefee;

        number = _number;
        timestamp = _timestamp;
        basefee = _averageBasefee();
        hash = _hash;
        sequenceNumber = _sequenceNumber;
    }

    /**
     * @notice Compute the average L1 basefee
     * Any calls to `setL1BlockValues` cannot revert, otherwise
     * the sequencer can manipulate the L1 context variables.
     * If the L1 basefee is very high, reverting transactions will make it be
     * stuck at a very large value which is bad for users.
     * Do unchecked math here so that an overflow doesn't result
     * in a revert. An overflow will only hurt the sequencer by making
     * the basefee that is used to charge users the L1 portion of their
     * transaction fee be much smaller.
     * Ignore samples that are not yet set to any values to prevent the
     * basefee from being extremely low before the basefees array is full
     */

    function _averageBasefee() internal returns (uint256) {
        unchecked {
            uint256 sum = 0;
            uint256 samples = 0;
            for (uint256 i = 0; i < AVERAGE_BASEFEE_WINDOW; i++) {
                uint256 sample = basefees[i];
                if (sample != 0) {
                    sum += basefees[i];
                    ++samples;
                }
            }
            if (samples == 0) {
                return 0;
            }
            // TODO: switch to solmate FixedPointMathLib.divWadUp
            // when relicensed to MIT
            return sum / samples;
        }
    }
}
