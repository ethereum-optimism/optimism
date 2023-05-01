// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeBuilder } from "./SafeBuilder.sol";
import { AddressManager } from "../../contracts/legacy/AddressManager.sol";
import { ProxyAdmin } from "../../contracts/universal/ProxyAdmin.sol";
import { L1ChugSplashProxy } from "../../contracts/legacy/L1ChugSplashProxy.sol";
import { Proxy } from "../../contracts/universal/Proxy.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { SystemDictator } from "../../contracts/deployment/SystemDictator.sol";
import { JSONConfig } from "./JSONConfig.sol";

contract BedrockUpgrade is SafeBuilder {
    /**
     * @notice A set of contracts for this upgrade transaction.
     */
    struct ContractSet {
        address systemDictator;
    }

    /**
     * @notice A json config files.
     */
    JSONConfig internal config;

    /**
     * @notice A mapping of chainid to a ContractSet of implementations.
     */
    mapping(uint256 => ContractSet) internal _implementations;

    /**
     * @notice A mapping of chainid to ContractSet of proxy addresses.
     */
    mapping(uint256 => ContractSet) internal proxies;

    function setUp() external {
        _implementations[MAINNET] = ContractSet({
            systemDictator: address(0xd6322f9d48439103d2e9c3bdA7A43F851FbB2423)
        });
    }

    function run(string memory _config, address _safe, address _proxyAdmin) external {
        config = new JSONConfig(_config);
        _run(_safe, _proxyAdmin);
    }

    function buildCalldata(address) internal override view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](2);

        ContractSet memory contractSet = implementations();

        // Call step1
        calls[0] = IMulticall3.Call3({
            target: contractSet.systemDictator,
            allowFailure: false,
            callData: abi.encodeCall(SystemDictator.step1, ())
        });

        // Call step2
        calls[1] = IMulticall3.Call3({
            target: contractSet.systemDictator,
            allowFailure: false,
            callData: abi.encodeCall(SystemDictator.step2, ())
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    function implementations() public view returns (ContractSet memory) {
        ContractSet memory cs = _implementations[block.chainid];
        require(cs.systemDictator != address(0), "implementations not set");
        return cs;
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal override view {
        ContractSet memory contractSet = implementations();
        require(SystemDictator(contractSet.systemDictator).currentStep() == uint8(3), "SystemDictator");
    }

    function test_script_succeeds() skipWhenNotForking external {
        address safe;
        address proxyAdmin;

        if (block.chainid == GOERLI) {
            safe = 0xBc1233d0C3e6B5d53Ab455cF65A6623F6dCd7e4f;
            proxyAdmin = 0x01d3670863c3F4b24D7b107900f0b75d4BbC6e0d;
        }

        require(safe != address(0) && proxyAdmin != address(0));

        address[] memory owners = IGnosisSafe(payable(safe)).getOwners();

        for (uint256 i; i < owners.length; i++) {
            address owner = owners[i];
            vm.startBroadcast(owner);
            bool success = _run(safe, proxyAdmin);
            vm.stopBroadcast();

            if (success) {
                console.log("tx success");
                break;
            }
        }

        _postCheck();
    }
}
