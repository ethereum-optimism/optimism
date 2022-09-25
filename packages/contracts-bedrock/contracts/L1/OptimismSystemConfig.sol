// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";
import { OptimismPortal } from "./OptimismPortal.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { GasPriceOracle } from "../L2/GasPriceOracle.sol";

//
contract OptimismSystemConfig is OwnableUpgradeable, Semver {

    uint256 public overhead;
    uint256 public scalar;

    OptimismPortal public portal;

    event OverheadUpdated(uint256 overhead);
    event ScalarUpdated(uint256 scalar);

    constructor(
        OptimismPortal _portal,
        uint256 _overhead,
        uint256 _scalar
    ) Semver(0, 0, 1) {
        portal = _portal;
        overhead = _overhead;
        scalar = _scalar;

        initialize(_portal, _overhead, _scalar);
    }

    function initialize(
        OptimismPortal _portal,
        uint256 _overhead,
        uint256 _scalar
    ) public {
        portal = _portal;
        overhead = _overhead;
        scalar = _scalar;
    }

    // set batcher key
    // set p2p key

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
