// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeBuilder } from "./SafeBuilder.sol";
import { AddressManager } from "../../contracts/legacy/AddressManager.sol";
import { ProxyAdmin } from "../../contracts/universal/ProxyAdmin.sol";
import { Proxy } from "../../contracts/universal/Proxy.sol";
import { L1ChugSplashProxy } from "../../contracts/legacy/L1ChugSplashProxy.sol";
import { Proxy } from "../../contracts/universal/Proxy.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

contract BedrockUpgrade is SafeBuilder {
    /**
     * @notice A set of contracts for this upgrade transaction.
     */
    struct ContractSet {
        address proxyAdmin;
        address addressManager;
        address l1StandardBridgeProxy;
        address l1ERC721BridgeProxy;
        address systemDicator;
    }

    /**
     * @notice A mapping of chainid to a ContractSet of implementations.
     */
    mapping(uint256 => ContractSet) internal _implementations;

    /**
     * @notice Sets up the ContractSets for the bedrock upgrade.
     */
    function setUp() external {
        _implementations[MAINNET] = ContractSet({
            proxyAdmin: address(0x6486ff256710d098a35f75b6fd0f4a5a1e045ec3),
            addressManager: address(0x6486ff256710D098a35F75b6FD0F4A5a1e045Ec3),
            l1StandardBridgeProxy: address(0x2907b87d7b7b27f60b37b57ff9156b752b419900),
            l1ERC721BridgeProxy: address(0xeFDd8B586cB777134216D54a6f344AFA5871e0d7),
            systemDicator: address(0xd6322f9d48439103d2e9c3bdA7A43F851FbB2423)
        });
    }

    function buildCalldata(address) internal override view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](3);

        ContractSet memory contractSet = implementations();

        // Transfer ownership of AddressManager to the systemDicator
        calls[0] = IMulticall3.Call3({
            target: contractSet.addressManager,
            allowFailure: false,
            callData: abi.encodeCall(
                Ownable.transferOwnership,
                (contractSet.systemDicator)
            )
        });

        // Transfer ownership of l1StandardBridgeProxy to the systemDicator
        calls[1] = IMulticall3.Call3({
            target: contractSet.l1StandardBridgeProxy,
            allowFailure: false,
            callData: abi.encodeCall(
                L1ChugSplashProxy.setOwner,
                (contractSet.systemDicator)
            )
        });

        // Transfer ownership of the l1ERC721BridgeProxy to the systemDicator
        calls[2] = IMulticall3.Call3({
            target: contractSet.l1ERC721BridgeProxy,
            allowFailure: false,
            callData: abi.encodeCall(
                Proxy.changeAdmin,
                (contractSet.systemDicator)
            )
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    function implementations() public view returns (ContractSet memory) {
        ContractSet memory cs = _implementations[block.chainid];
        require(cs.proxyAdmin != address(0), "implementations not set");
        return cs;
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal override view {
        ContractSet memory contractSet = implementations();
        require(Ownable(contractSet.addressManager).owner() == contractSet.systemDictator, "AddressManager");
        require(L1ChugSplashProxy(contractSet.l1StandardBridgeProxy).getOwner() == contractSet.systemDicator, "l1StandardBridgeProxy");
        require(Proxy(contractSet.l1ERC721BridgeProxy).admin() == contractSet.systemDicator, "l1ERC721BridgeProxy");
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
