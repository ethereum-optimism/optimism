pragma solidity 0.8.15;

import { Burn } from "../libraries/Burn.sol";
import { StdUtils } from "forge-std/Test.sol";

contract EchidnaFuzzBurnEth is StdUtils {
    bool internal failedEthBurn;

    /**
     * @notice Takes an integer amount of eth to burn through the Burn library and
     * updates the contract state if an incorrect amount of eth moved from the contract
     */
    function testBurn(uint256 _value) public {
        // cache the contract's eth balance
        uint256 preBurnBalance = address(this).balance;
        uint256 value = bound(_value, 0, preBurnBalance);

        // execute a burn of _value eth
        Burn.eth(value);

        // check that exactly value eth was transfered from the contract
        unchecked {
            if (address(this).balance != preBurnBalance - value) {
                failedEthBurn = true;
            }
        }
    }

    /**
     * @custom:invariant `eth(uint256)` always burns the exact amount of eth passed.
     *
     * Asserts that when `Burn.eth(uint256)` is called, it always burns the exact amount
     * of ETH passed to the function.
     */
    function echidna_burn_eth() public view returns (bool) {
        // ASSERTION: The amount burned should always match the amount passed exactly
        return !failedEthBurn;
    }
}

contract EchidnaFuzzBurnGas is StdUtils {
    bool internal failedGasBurn;

    /**
     * @notice Takes an integer amount of gas to burn through the Burn library and
     * updates the contract state if at least that amount of gas was not burned
     * by the library
     */
    function testGas(uint256 _value) public {
        // cap the value to the max resource limit
        uint256 MAX_RESOURCE_LIMIT = 8_000_000;
        uint256 value = bound(_value, 0, MAX_RESOURCE_LIMIT);

        // cache the contract's current remaining gas
        uint256 preBurnGas = gasleft();

        // execute the gas burn
        Burn.gas(value);

        // cache the remaining gas post burn
        uint256 postBurnGas = gasleft();

        // check that at least value gas was burnt (and that there was no underflow)
        unchecked {
            if (postBurnGas - preBurnGas > value || preBurnGas - value > preBurnGas) {
                failedGasBurn = true;
            }
        }
    }

    /**
     * @custom:invariant `gas(uint256)` always burns at least the amount of gas passed.
     *
     * Asserts that when `Burn.gas(uint256)` is called, it always burns at least the amount
     * of gas passed to the function.
     */
    function echidna_burn_gas() public view returns (bool) {
        // ASSERTION: The amount of gas burned should be strictly greater than the
        // the amount passed as _value (minimum _value + whatever minor overhead to
        // the value after the call)
        return !failedGasBurn;
    }
}
