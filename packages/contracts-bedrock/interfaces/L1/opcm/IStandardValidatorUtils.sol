// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import {
    DisputeGameValidationArgs,
    DisputeGameImpls,
    DisputeGameConfig,
    SuperPermissionedDisputeGameValidationArgs,
    SuperPermissionedDisputeGameImpls
} from "src/L1/opcm/StandardValidatorUtils.sol";

interface IStandardValidatorUtils {
    function __constructor__() external;

    function assertValidSuperchainConfig(
        string memory _errors,
        ISuperchainConfig _superchainConfig
    )
        external
        view
        returns (string memory);

    function assertValidProxyAdmin(
        string memory _errors,
        IProxyAdmin _admin,
        address _l1PAOMultisig
    )
        external
        view
        returns (string memory);

    function assertValidSuperRootDisputeGames(
        string memory _errors,
        ISystemConfig _sysCfg
    )
        external
        view
        returns (string memory);

    function assertValidNonSuperRootDisputeGames(
        string memory _errors,
        ISystemConfig _sysCfg
    )
        external
        view
        returns (string memory);

    function assertValidMipsVm(
        string memory _errors,
        IMIPS64 _mips,
        address _mipsImpl,
        string memory _errorPrefix
    )
        external
        view
        returns (string memory);

    function assertValidOptimismMintableERC20Factory(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        address _impl
    )
        external
        view
        returns (string memory);

    function assertValidDisputeGameFactory(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        address _impl,
        address _l1PAOMultisig
    )
        external
        view
        returns (string memory);

    function assertValidOptimismPortal(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        address _impl
    )
        external
        view
        returns (string memory);

    function assertValidPreimageOracle(
        string memory _errors,
        IPreimageOracle _oracle,
        string memory _errorPrefix
    )
        external
        view
        returns (string memory);

    function assertValidDelayedWETH(
        string memory _errors,
        ISystemConfig _sysCfg,
        IDelayedWETH _weth,
        IProxyAdmin _admin,
        address _l1PAOMultisig,
        address _delayedWETHImpl,
        uint256 _withdrawalDelaySeconds,
        string memory _errorPrefix
    )
        external
        view
        returns (string memory);

    function assertValidAnchorStateRegistry(
        string memory _errors,
        ISystemConfig _sysCfg,
        IDisputeGameFactory _dgf,
        IAnchorStateRegistry _asr,
        IProxyAdmin _admin,
        address _anchorStateRegistryImpl,
        string memory _errorPrefix
    )
        external
        view
        returns (string memory);

    function assertValidDisputeGame(
        DisputeGameValidationArgs memory _args,
        DisputeGameImpls memory _impls,
        DisputeGameConfig memory _cfg
    )
        external
        view
        returns (string memory errors_);

    function assertValidSuperPermissionedDisputeGame(
        SuperPermissionedDisputeGameValidationArgs memory _args,
        SuperPermissionedDisputeGameImpls memory _impls
    )
        external
        view
        returns (string memory errors_);
}
