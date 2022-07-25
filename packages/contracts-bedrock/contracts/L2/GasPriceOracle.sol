// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { L1Block } from "../L2/L1Block.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x420000000000000000000000000000000000000F
 * @title GasPriceOracle
 * @notice This contract maintains the variables responsible for computing the L1 portion of the
 *         total fee charged on L2. The values stored in the contract are looked up as part of the
 *         L2 state transition function and used to compute the total fee paid by the user. The
 *         contract exposes an API that is useful for knowing how large the L1 portion of their
 *         transaction fee will be.
 */
contract GasPriceOracle is Ownable, Semver {
    /**
     * @custom:legacy
     * @notice Spacer for backwards compatibility.
     */
    uint256 internal spacer0;

    /**
     * @custom:legacy
     * @notice Spacer for backwards compatibility.
     */
    uint256 internal spacer1;

    /**
     * @notice Constant L1 gas overhead per transaction.
     */
    uint256 public overhead;

    /**
     * @notice Dynamic L1 gas overhead per transaction.
     */
    uint256 public scalar;

    /**
     * @notice Number of decimals used in the scalar.
     */
    uint256 public decimals;

    /**
     * @notice Emitted when the overhead value is updated.
     */
    event OverheadUpdated(uint256 overhead);

    /**
     * @notice Emitted when the scalar value is updated.
     */
    event ScalarUpdated(uint256 scalar);

    /**
     * @notice Emitted when the decimals value is updated.
     */
    event DecimalsUpdated(uint256 decimals);

    /**
     * @custom:semver 0.0.1
     *
     * @param _owner Address that will initially own this contract.
     */
    constructor(address _owner) Ownable() Semver(0, 0, 1) {
        transferOwnership(_owner);
    }

    /**
     * @notice Allows the owner to modify the overhead.
     *
     * @param _overhead New overhead value.
     */
    function setOverhead(uint256 _overhead) external onlyOwner {
        overhead = _overhead;
        emit OverheadUpdated(_overhead);
    }

    /**
     * @notice Allows the owner to modify the scalar.
     *
     * @param _scalar New scalar value.
     */
    function setScalar(uint256 _scalar) external onlyOwner {
        scalar = _scalar;
        emit ScalarUpdated(_scalar);
    }

    /**
     * @notice Allows the owner to modify the decimals.
     *
     * @param _decimals New decimals value.
     */
    function setDecimals(uint256 _decimals) external onlyOwner {
        decimals = _decimals;
        emit DecimalsUpdated(_decimals);
    }

    /**
     * @notice Computes the L1 portion of the fee based on the size of the rlp encoded input
     *         transaction, the current L1 base fee, and the various dynamic parameters.
     *
     * @param _data Unsigned fully RLP-encoded transaction to get the L1 fee for.
     *
     * @return L1 fee that should be paid for the tx
     */
    function getL1Fee(bytes memory _data) external view returns (uint256) {
        uint256 l1GasUsed = getL1GasUsed(_data);
        uint256 l1Fee = l1GasUsed * l1BaseFee();
        uint256 divisor = 10**decimals;
        uint256 unscaled = l1Fee * scalar;
        uint256 scaled = unscaled / divisor;
        return scaled;
    }

    /**
     * @notice Retrieves the current gas price (base fee).
     *
     * @return Current L2 gas price (base fee).
     */
    function gasPrice() public view returns (uint256) {
        return block.basefee;
    }

    /**
     * @notice Retrieves the current base fee.
     *
     * @return Current L2 base fee.
     */
    function baseFee() public view returns (uint256) {
        return block.basefee;
    }

    /**
     * @notice Retrieves the latest known L1 base fee.
     *
     * @return Latest known L1 base fee.
     */
    function l1BaseFee() public view returns (uint256) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).basefee();
    }

    /**
     * @notice Computes the amount of L1 gas used for a transaction. Adds the overhead which
     *         represents the per-transaction gas overhead of posting the transaction and state
     *         roots to L1. Adds 68 bytes of padding to account for the fact that the input does
     *         not have a signature.
     *
     * @param _data Unsigned fully RLP-encoded transaction to get the L1 gas for.
     *
     * @return Amount of L1 gas used to publish the transaction.
     */
    function getL1GasUsed(bytes memory _data) public view returns (uint256) {
        uint256 total = 0;
        uint256 length = _data.length;
        for (uint256 i = 0; i < length; i++) {
            if (_data[i] == 0) {
                total += 4;
            } else {
                total += 16;
            }
        }
        uint256 unsigned = total + overhead;
        return unsigned + (68 * 16);
    }
}
