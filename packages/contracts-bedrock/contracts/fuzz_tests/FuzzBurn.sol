import { Burn } from "../libraries/Burn.sol";

contract FuzzBurn {
    bool failedEthBurn;
    bool failedGasBurn;

    /**
     * @notice Takes an integer amount of eth to burn through the Burn library and
     * updates the contract state if an incorrect amount of eth moved from the contract
     */
    function testBurn(uint256 _value) public {
        // cache the contract's eth balance
        uint256 preBurnBalance = address(this).balance;

        // execute a burn of _value eth
        // (may way to add guardrails to this value rather than a truly unbounded uint256)
        Burn.eth(_value);

        // check that exactly _value eth was transfered from the contract
        if (address(this).balance != preBurnBalance - _value) {
            failedEthBurn = true;
        }
    }

    /**
     * @notice Takes an integer amount of gas to burn through the Burn library and
     * updates the contract state if at least that amount of gas was not burned
     * by the library
     */
    function testGas(uint256 _value) public {
        // cache the contract's current remaining gas
        uint256 preBurnGas = gasleft();

        // execute the gas burn
        Burn.gas(_value);

        // cache the remaining gas post burn
        uint256 postBurnGas = gasleft();

        // check that at least _value gas was burnt
        if (postBurnGas > preBurnGas - _value) {
            failedGasBurn;
        }
    }

    function echidna_burn_eth() public view returns(bool) {
        // ASSERTION: The amount burned should always match the amount passed exactly
        return !failedEthBurn;
    }

    function echidna_burn_gas() public view returns (bool) {
        // ASSERTION: The amount of gas burned should be strictly greater than the
        // the amount passed as _value (minimum _value + whatever minor overhead to
        // the value after the call)
        return !failedGasBurn;
    }
}