// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";
import { OptimismPortal } from "./OptimismPortal.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { GasPriceOracle } from "../L2/GasPriceOracle.sol";

/**
 * @title OptimismSystemConfig
 * @notice This contract...
 *
 *         TODO:
 *           - set batcher key
 *           - set p2p key
 */
contract OptimismSystemConfig is OwnableUpgradeable, Semver {
    /**
     *
     */
    uint256 public overhead;

    /**
     *
     */
    uint256 public scalar;

    /**
     *
     */
    OptimismPortal public portal;

    /**
     *
     */
    event OverheadUpdated(uint256 overhead);

    /**
     *
     */
    event ScalarUpdated(uint256 scalar);

    /**
     * @custom:semver 0.0.1
     *
     * @param _owner Address that will initially own this contract.
     * @param _portal Address of the OptimismPortal
     * @param _overhead Initial value of GasPriceOracle overhead
     * @param _scalar Initial value of the GasPriceOracle scalar
     */
    constructor(
        address _owner,
        OptimismPortal _portal,
        uint256 _overhead,
        uint256 _scalar
    ) Semver(0, 0, 1) {
        initialize(_owner, _portal, _overhead, _scalar);
    }

    /**
     * @notice
     */
    function initialize(
        address _owner,
        OptimismPortal _portal,
        uint256 _overhead,
        uint256 _scalar
    ) public initializer {
        portal = _portal;
        overhead = _overhead;
        scalar = _scalar;

        _transferOwnership(_owner);
    }

    /**
     * @notice
     */
    function setOverhead(uint256 _overhead) external {
        overhead = _overhead;

        _callOptimismPortal(
            Predeploys.GAS_PRICE_ORACLE,
            abi.encodeWithSelector(
                GasPriceOracle.setOverhead.selector,
                abi.encode(_overhead)
            )
        );

        emit ScalarUpdated(_overhead);
    }

    /**
     * @notice
     */
    function setScalar(uint256 _scalar) external {
        scalar = _scalar;

        _callOptimismPortal(
            Predeploys.GAS_PRICE_ORACLE,
            abi.encodeWithSelector(
                GasPriceOracle.setScalar.selector,
                abi.encode(_scalar)
            )
        );

        emit OverheadUpdated(_scalar);
    }

    /**
     *
     */
    function _callOptimismPortal(address _to, bytes memory _data) internal {
        portal.depositTransaction(
            _to,
            0,
            10000,
            false,
            _data
        );
    }
}
