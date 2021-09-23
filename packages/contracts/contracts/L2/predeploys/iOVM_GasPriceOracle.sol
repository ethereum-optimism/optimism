// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/**
 * @title iOVM_GasPriceOracle
 */
interface iOVM_GasPriceOracle {

    /**********
     * Events *
     **********/

    event GasPriceUpdated(uint256);
    event L1BaseFeeUpdated(uint256);
    event OverheadUpdated(uint256);
    event ScalarUpdated(uint256);
    event DecimalsUpdated(uint256);

    /********************
     * Public Functions *
     ********************/

    function setGasPrice(uint256 _gasPrice) external;
    function setL1BaseFee(uint256 _baseFee) external;
    function setOverhead(uint256 _overhead) external;
    function setScalar(uint256 _scalar) external;
    function setDecimals(uint256 _decimals) external;
    function getL1Fee(bytes memory _data) external returns (uint256);
    function getL1GasUsed(bytes memory _data) external returns (uint256);
}
