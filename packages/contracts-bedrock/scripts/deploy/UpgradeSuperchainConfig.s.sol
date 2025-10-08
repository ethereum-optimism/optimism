// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";

contract UpgradeSuperchainConfigInput is BaseDeployIO {
    address internal _prank;
    IOPContractsManager internal _opcm;
    ISuperchainConfig internal _superchainConfig;
    IProxyAdmin internal _superchainProxyAdmin;

    // Setter for OPContractsManager type
    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "UpgradeSuperchainConfigInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = IOPContractsManager(_value);
        else if (_sel == this.superchainConfig.selector) _superchainConfig = ISuperchainConfig(_value);
        else if (_sel == this.superchainProxyAdmin.selector) _superchainProxyAdmin = IProxyAdmin(_value);
        else revert("UpgradeSuperchainConfigInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "UpgradeSuperchainConfigInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (IOPContractsManager) {
        require(address(_opcm) != address(0), "UpgradeSuperchainConfigInput: opcm not set");
        return _opcm;
    }

    function superchainConfig() public view returns (ISuperchainConfig) {
        require(address(_superchainConfig) != address(0), "UpgradeSuperchainConfigInput: superchainConfig not set");
        return _superchainConfig;
    }

    function superchainProxyAdmin() public view returns (IProxyAdmin) {
        require(
            address(_superchainProxyAdmin) != address(0), "UpgradeSuperchainConfigInput: superchainProxyAdmin not set"
        );
        return _superchainProxyAdmin;
    }
}

contract UpgradeSuperchainConfig is Script {
    function run(UpgradeSuperchainConfigInput _uoci) external {
        IOPContractsManager opcm = _uoci.opcm();

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _uoci.prank();

        bytes memory code = vm.getDeployedCode("UpgradeSuperchainConfig.s.sol:DummyCaller");
        vm.etch(prank, code);

        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
        vm.label(prank, "DummyCaller");

        ISuperchainConfig superchainConfig = _uoci.superchainConfig();
        IProxyAdmin superchainProxyAdmin = _uoci.superchainProxyAdmin();

        // Call into the DummyCaller to perform the delegatecall
        vm.broadcast(msg.sender);

        (bool success,) = DummyCaller(prank).upgradeSuperchainConfig(superchainConfig, superchainProxyAdmin);
        require(success, "UpgradeSuperchainConfig: upgradeSuperchainConfig failed");
    }
}

/// @title DummyCaller
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
contract DummyCaller {
    address internal _opcmAddr;

    function upgradeSuperchainConfig(
        ISuperchainConfig _superchainConfig,
        IProxyAdmin _superchainProxyAdmin
    )
        external
        returns (bool, bytes memory)
    {
        bytes memory data =
            abi.encodeCall(DummyCaller.upgradeSuperchainConfig, (_superchainConfig, _superchainProxyAdmin));
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}
