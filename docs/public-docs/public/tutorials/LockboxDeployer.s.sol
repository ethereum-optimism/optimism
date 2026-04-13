// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import {Script, console} from "forge-std/Script.sol";
import {Vm} from "forge-std/Vm.sol";
import {LockboxSuperchainERC20} from "../src/LockboxSuperchainERC20.sol";

contract LockboxDeployer is Script {
    string deployConfig;
    uint256 timestamp;

    constructor() {
        string memory deployConfigPath = vm.envOr("DEPLOY_CONFIG_PATH", string("/configs/deploy-config.toml"));
        string memory filePath = string.concat(vm.projectRoot(), deployConfigPath);
        deployConfig = vm.readFile(filePath);
        timestamp = vm.unixTime();
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast(msg.sender);
        _;
        vm.stopBroadcast();
    }

    function setUp() public {}

    function run() public {
        string[] memory chainsToDeployTo = vm.parseTomlStringArray(deployConfig, ".deploy_config.chains");

        address deployedAddress;

        for (uint256 i = 0; i < chainsToDeployTo.length; i++) {
            string memory chainToDeployTo = chainsToDeployTo[i];

            console.log("Deploying to chain: ", chainToDeployTo);

            vm.createSelectFork(chainToDeployTo);
            address _deployedAddress = deployLockboxSuperchainERC20();
            deployedAddress = _deployedAddress;
        }

        outputDeploymentResult(deployedAddress);
    }

    function deployLockboxSuperchainERC20() public broadcast returns (address addr_) {
        string memory name = vm.envString("NEW_TOKEN_NAME");
        string memory symbol = vm.envString("NEW_TOKEN_SYMBOL");
        uint256 decimals = vm.envUint("TOKEN_DECIMALS");
        require(decimals <= type(uint8).max, "decimals exceeds uint8 range");
        address originalTokenAddress = vm.envAddress("ERC20_ADDRESS");
        uint256 originalChainId = vm.envUint("ERC20_CHAINID");

        bytes memory initCode = abi.encodePacked(
            type(LockboxSuperchainERC20).creationCode, 
                 abi.encode(name, symbol, uint8(decimals), originalTokenAddress, originalChainId)
        );
        address preComputedAddress = vm.computeCreate2Address(_implSalt(), keccak256(initCode));
        if (preComputedAddress.code.length > 0) {
            console.log(
                "There is already a contract at %s", preComputedAddress, "on chain id: ", block.chainid
            );
            addr_ = preComputedAddress;
        } else {
            addr_ = address(new LockboxSuperchainERC20{salt: _implSalt()}(
                name, symbol, uint8(decimals), originalTokenAddress, originalChainId));
            console.log("Deployed LockboxSuperchainERC20 at address: ", addr_, "on chain id: ", block.chainid);
        }
    }

    function outputDeploymentResult(address deployedAddress) public {
        console.log("Outputting deployment result");

        string memory obj = "result";
        string memory jsonOutput = vm.serializeAddress(obj, "deployedAddress", deployedAddress);

        vm.writeJson(jsonOutput, "deployment.json");
    }

    /// @notice The CREATE2 salt to be used when deploying the token.
    function _implSalt() internal view returns (bytes32) {
        string memory salt = vm.parseTomlString(deployConfig, ".deploy_config.salt");
        return keccak256(abi.encodePacked(salt, timestamp));
    }
}
