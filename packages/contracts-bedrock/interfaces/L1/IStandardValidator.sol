// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

interface IStandardValidator {
    struct Implementations {
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
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

    function anchorStateRegistryImpl() external view returns (address);
    function anchorStateRegistryVersion() external pure returns (string memory);
    function challenger() external view returns (address);
    function delayedWETHImpl() external view returns (address);
    function delayedWETHVersion() external pure returns (string memory);
    function disputeGameFactoryImpl() external view returns (address);
    function disputeGameFactoryVersion() external pure returns (string memory);
    function l1CrossDomainMessengerImpl() external view returns (address);
    function l1CrossDomainMessengerVersion() external pure returns (string memory);
    function l1ERC721BridgeImpl() external view returns (address);
    function l1ERC721BridgeVersion() external pure returns (string memory);
    function l1PAOMultisig() external view returns (address);
    function l1StandardBridgeImpl() external view returns (address);
    function l1StandardBridgeVersion() external pure returns (string memory);
    function mipsImpl() external view returns (address);
    function mipsVersion() external pure returns (string memory);
    function optimismMintableERC20FactoryImpl() external view returns (address);
    function optimismMintableERC20FactoryVersion() external pure returns (string memory);
    function optimismPortalImpl() external view returns (address);
    function optimismPortalVersion() external pure returns (string memory);
    function permissionedDisputeGameVersion() external pure returns (string memory);
    function preimageOracleVersion() external pure returns (string memory);
    function superchainConfig() external view returns (ISuperchainConfig);
    function systemConfigImpl() external view returns (address);
    function systemConfigVersion() external pure returns (string memory);
    function withdrawalDelaySeconds() external view returns (uint256);

    function validate(
        ValidationInput memory _input,
        bool _allowFailure,
        ValidationOverrides memory _overrides
    )
        external
        view
        returns (string memory);
    function validate(ValidationInput memory _input, bool _allowFailure) external view returns (string memory);

    function __constructor__(
        IStandardValidator.Implementations memory _implementations,
        ISuperchainConfig _superchainConfig,
        address _l1PAOMultisig,
        address _challenger,
        uint256 _withdrawalDelaySeconds
    )
        external;
}
