// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

interface IOPContractsManagerStandardValidator {
    struct Implementations {
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
        address optimismPortalInteropImpl;
        address ethLockboxImpl;
        address systemConfigImpl;
        address optimismMintableERC20FactoryImpl;
        address l1CrossDomainMessengerImpl;
        address l1StandardBridgeImpl;
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address delayedWETHImpl;
        address mipsImpl;
    }

    struct ValidationInput {
        IProxyAdmin proxyAdmin;
        ISystemConfig sysCfg;
        bytes32 absolutePrestate;
        uint256 l2ChainID;
    }

    struct ValidationOverrides {
        address l1PAOMultisig;
        address challenger;
    }

    function version() external view returns (string memory);
    function anchorStateRegistryImpl() external view returns (address);
    function challenger() external view returns (address);
    function delayedWETHImpl() external view returns (address);
    function devFeatureBitmap() external view returns (bytes32);
    function disputeGameFactoryImpl() external view returns (address);
    function l1CrossDomainMessengerImpl() external view returns (address);
    function l1ERC721BridgeImpl() external view returns (address);
    function l1PAOMultisig() external view returns (address);
    function l1StandardBridgeImpl() external view returns (address);
    function mipsImpl() external view returns (address);
    function optimismMintableERC20FactoryImpl() external view returns (address);
    function optimismPortalImpl() external view returns (address);
    function optimismPortalInteropImpl() external view returns (address);
    function ethLockboxImpl() external view returns (address);
    function permissionedDisputeGameVersion() external pure returns (string memory);
    function preimageOracleVersion() external pure returns (string memory);
    function superchainConfig() external view returns (ISuperchainConfig);
    function systemConfigImpl() external view returns (address);
    function withdrawalDelaySeconds() external view returns (uint256);

    function validateWithOverrides(
        ValidationInput memory _input,
        bool _allowFailure,
        ValidationOverrides memory _overrides
    )
        external
        view
        returns (string memory);
    function validate(ValidationInput memory _input, bool _allowFailure) external view returns (string memory);

    function __constructor__(
        Implementations memory _implementations,
        ISuperchainConfig _superchainConfig,
        address _l1PAOMultisig,
        address _challenger,
        uint256 _withdrawalDelaySeconds,
        bytes32 _devFeatureBitmap
    )
        external;
}
