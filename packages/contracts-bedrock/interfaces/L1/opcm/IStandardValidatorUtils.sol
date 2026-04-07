// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

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
}
