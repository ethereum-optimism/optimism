// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { Script } from "forge-std/Script.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { IGnosisSafe } from "./IGnosisSafe.sol";
import { LibSort } from "./LibSort.sol";
import { Enum } from "./Enum.sol";
import { ProxyAdmin } from "../../contracts/universal/ProxyAdmin.sol";
import { Constants } from "../../contracts/libraries/Constants.sol";
import { SystemConfig } from "../../contracts/L1/SystemConfig.sol";
import { ResourceMetering } from "../../contracts/L1/ResourceMetering.sol";
import { Semver } from "../../contracts/universal/Semver.sol";

/**
 * @title PostSherlock
 * @notice Upgrade script for upgrading the L1 contracts after the sherlock audit.
 *         Assumes that a gnosis safe is used as the privileged account and the same
 *         gnosis safe is the owner of the system config and the proxy admin.
 *         This could be optimized by checking for the number of approvals up front
 *         and not submitting the final approval as `execTransaction` can be called when
 *         there are `threshold - 1` approvals.
 *         Uses the "approved hashes" method of interacting with the gnosis safe. Allows
 *         for the most simple user experience when using automation and no indexer.
 *         Run the command without the `--broadcast` flag and it will print a tenderly URL.
 */
contract PostSherlock is Script {
    /**
     * @notice Interface for multicall3.
     */
    IMulticall3 private constant multicall = IMulticall3(MULTICALL3_ADDRESS);

    /**
     * @notice Mainnet chain id.
     */
    uint256 constant MAINNET = 1;

    /**
     * @notice Goerli chain id.
     */
    uint256 constant GOERLI = 5;

    /**
     * @notice Represents a set of L1 contracts. Used to represent a set of
     *         implementations and also a set of proxies.
     */
    struct ContractSet {
        address L1CrossDomainMessenger;
        address L1StandardBridge;
        address L2OutputOracle;
        address OptimismMintableERC20Factory;
        address OptimismPortal;
        address SystemConfig;
        address L1ERC721Bridge;
    }

    /**
     * @notice A mapping of chainid to a ContractSet of implementations.
     */
    mapping(uint256 => ContractSet) internal implementations;

    /**
     * @notice A mapping of chainid to ContractSet of proxy addresses.
     */
    mapping(uint256 => ContractSet) internal proxies;

    /**
     * @notice An array of approvals, used to generate the execution transaction.
     */
    address[] internal approvals;

    /**
     * @notice The expected versions for the contracts to be upgraded to.
     */
    string constant internal L1CrossDomainMessenger_Version = "1.1.0";
    string constant internal L1StandardBridge_Version = "1.1.0";
    string constant internal L2OutputOracle_Version = "1.2.0";
    string constant internal OptimismMintableERC20Factory_Version = "1.1.0";
    string constant internal OptimismPortal_Version = "1.3.1";
    string constant internal SystemConfig_Version = "1.2.0";
    string constant internal L1ERC721Bridge_Version = "1.1.0";

    /**
     * @notice Place the contract addresses in storage so they can be used when building calldata.
     */
    function setUp() external {
        implementations[GOERLI] = ContractSet({
            L1CrossDomainMessenger: 0xfa37a4b2D49E21De63fa2b13D6dB213081E020b3,
            L1StandardBridge: 0x79179704077E3324CC745A24a5CcC2a80A9B6842,
            L2OutputOracle: 0x47bBB9054823f27B9B6A71F5cb0eBc785692FF2E,
            OptimismMintableERC20Factory: 0xF516Fa87f89E4AC7C299aE28263e9EB851dE4781,
            OptimismPortal: 0xa24A444C6ceeb1d4Fc19D1B78913C22B9d03BbC9,
            SystemConfig: 0x2FFfe603caA9FA2C20E7F349138475a43284a6b1,
            L1ERC721Bridge: 0xb460323429B08B9d1d427e6b8A450532988d5fe8
        });

        proxies[GOERLI] = ContractSet({
            L1CrossDomainMessenger: 0x5086d1eEF304eb5284A0f6720f79403b4e9bE294,
            L1StandardBridge: 0x636Af16bf2f682dD3109e60102b8E1A089FedAa8,
            L2OutputOracle: 0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0,
            OptimismMintableERC20Factory: 0x883dcF8B05364083D849D8bD226bC8Cb4c42F9C5,
            OptimismPortal: 0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383,
            SystemConfig: 0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60,
            L1ERC721Bridge: 0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9
        });
    }

    /**
     * @notice The entrypoint to this script.
     */
    function run(address _safe, address _proxyAdmin) external returns (bool) {
        vm.startBroadcast();
        bool success = _run(_safe, _proxyAdmin);
        if (success) _postCheck();
        return success;
    }

    /**
     * @notice The implementation of the upgrade. Split into its own function
     *         to allow for testability. This is subject to a race condition if
     *         the nonce changes by a different transaction finalizing while not
     *         all of the signers have used this script.
     */
    function _run(address _safe, address _proxyAdmin) public returns (bool) {
        // Ensure that the required contracts exist
        require(address(multicall).code.length > 0, "multicall3 not deployed");
        require(_safe.code.length > 0, "no code at safe address");
        require(_proxyAdmin.code.length > 0, "no code at proxy admin address");

        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        uint256 nonce = safe.nonce();

        bytes memory data = buildCalldata(_proxyAdmin);

        // Compute the safe transaction hash
        bytes32 hash = safe.getTransactionHash({
            to: address(multicall),
            value: 0,
            data: data,
            operation: Enum.Operation.DelegateCall,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: address(0),
            _nonce: nonce
        });

        // Send a transaction to approve the hash
        safe.approveHash(hash);

        logSimulationLink({
            _to: address(safe),
            _from: msg.sender,
            _data: abi.encodeCall(safe.approveHash, (hash))
        });

        uint256 threshold = safe.getThreshold();
        address[] memory owners = safe.getOwners();

        for (uint256 i; i < owners.length; i++) {
            address owner = owners[i];
            uint256 approved = safe.approvedHashes(owner, hash);
            if (approved == 1) {
                approvals.push(owner);
            }
        }

        if (approvals.length >= threshold) {
            bytes memory signatures = buildSignatures();

            bool success = safe.execTransaction({
                to: address(multicall),
                value: 0,
                data: data,
                operation: Enum.Operation.DelegateCall,
                safeTxGas: 0,
                baseGas: 0,
                gasPrice: 0,
                gasToken: address(0),
                refundReceiver: payable(address(0)),
                signatures: signatures
            });

            logSimulationLink({
                _to: address(safe),
                _from: msg.sender,
                _data: abi.encodeCall(
                    safe.execTransaction,
                    (
                        address(multicall),
                        0,
                        data,
                        Enum.Operation.DelegateCall,
                        0,
                        0,
                        0,
                        address(0),
                        payable(address(0)),
                        signatures
                    )
                )
            });

            require(success, "call not successful");
            return true;
        } else {
            console.log("not enough approvals");
        }

        // Reset the approvals because they are only used transiently.
        assembly {
            sstore(approvals.slot, 0)
        }

        return false;
    }

    /**
     * @notice Log a tenderly simulation link. The TENDERLY_USERNAME and TENDERLY_PROJECT
     *         environment variables will be used if they are present. The vm is staticcall'ed
     *         because of a compiler issue with the higher level ABI.
     */
    function logSimulationLink(address _to, bytes memory _data, address _from) public view {
        (, bytes memory projData) = VM_ADDRESS.staticcall(
            abi.encodeWithSignature("envOr(string,string)", "TENDERLY_PROJECT", "TENDERLY_PROJECT")
        );
        string memory proj = abi.decode(projData, (string));

        (, bytes memory userData) = VM_ADDRESS.staticcall(
            abi.encodeWithSignature("envOr(string,string)", "TENDERLY_USERNAME", "TENDERLY_USERNAME")
        );
        string memory username = abi.decode(userData, (string));

        string memory str = string.concat(
            "https://dashboard.tenderly.co/",
            username,
            "/",
            proj,
            "/simulator/new?network=",
            vm.toString(block.chainid),
            "&contractAddress=",
            vm.toString(_to),
            "&rawFunctionInput=",
            vm.toString(_data),
            "&from=",
            vm.toString(_from)
        );
        console.log(str);
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal view {
        ContractSet memory prox = getProxies();
        require(_versionHash(prox.L1CrossDomainMessenger) == keccak256(bytes(L1CrossDomainMessenger_Version)));
        require(_versionHash(prox.L1StandardBridge) == keccak256(bytes(L1StandardBridge_Version)));
        require(_versionHash(prox.L2OutputOracle) == keccak256(bytes(L2OutputOracle_Version)));
        require(_versionHash(prox.OptimismMintableERC20Factory) == keccak256(bytes(OptimismMintableERC20Factory_Version)));
        require(_versionHash(prox.OptimismPortal) == keccak256(bytes(OptimismPortal_Version)));
        require(_versionHash(prox.SystemConfig) == keccak256(bytes(SystemConfig_Version)));
        require(_versionHash(prox.L1ERC721Bridge) == keccak256(bytes(L1ERC721Bridge_Version)));

        ResourceMetering.ResourceConfig memory rcfg = SystemConfig(prox.SystemConfig).resourceConfig();
        ResourceMetering.ResourceConfig memory dflt = Constants.DEFAULT_RESOURCE_CONFIG();
        require(keccak256(abi.encode(rcfg)) == keccak256(abi.encode(dflt)));
    }

    /**
     * @notice Helper function used to compute the hash of Semver's version string to be used in a
     *         comparison.
     */
    function _versionHash(address _addr) internal view returns (bytes32) {
        return keccak256(bytes(Semver(_addr).version()));
    }

    /**
     * @notice Test coverage of the logic. Should only run on goerli but other chains
     *         could be added.
     */
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

    /**
     * @notice Builds the signatures by tightly packing them together.
     *         Ensures that they are sorted.
     */
    function buildSignatures() internal view returns (bytes memory) {
        address[] memory addrs = new address[](approvals.length);
        for (uint256 i; i < approvals.length; i++) {
            addrs[i] = approvals[i];
        }

        LibSort.sort(addrs);

        bytes memory signatures;
        uint8 v = 1;
        bytes32 s = bytes32(0);
        for (uint256 i; i < addrs.length; i++) {
            bytes32 r = bytes32(uint256(uint160(addrs[i])));
            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }
        return signatures;
    }

    /**
     * @notice Builds the calldata that the multisig needs to make for the upgrade to happen.
     *         A total of 8 calls are made, 7 upgrade implementations and 1 sets the resource
     *         config to the default value in the SystemConfig contract.
     */
    function buildCalldata(address _proxyAdmin) internal view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](8);

        ContractSet memory impl = getImplementations();
        ContractSet memory prox = getProxies();

        // Upgrade the L1CrossDomainMessenger
        calls[0] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1CrossDomainMessenger), impl.L1CrossDomainMessenger)
            )
        });

        // Upgrade the L1StandardBridge
        calls[1] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1StandardBridge), impl.L1StandardBridge)
            )
        });

        // Upgrade the L2OutputOracle
        calls[2] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2OutputOracle), impl.L2OutputOracle)
            )
        });

        // Upgrade the OptimismMintableERC20Factory
        calls[3] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismMintableERC20Factory), impl.OptimismMintableERC20Factory)
            )
        });

        // Upgrade the OptimismPortal
        calls[4] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismPortal), impl.OptimismPortal)
            )
        });

        // Upgrade the SystemConfig
        calls[5] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.SystemConfig), impl.SystemConfig)
            )
        });

        // Upgrade the L1ERC721Bridge
        calls[6] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1ERC721Bridge), impl.L1ERC721Bridge)
            )
        });

        // Set the default resource config
        ResourceMetering.ResourceConfig memory rcfg = Constants.DEFAULT_RESOURCE_CONFIG();
        calls[7] = IMulticall3.Call3({
            target: prox.SystemConfig,
            allowFailure: false,
            callData: abi.encodeCall(SystemConfig.setResourceConfig, (rcfg))
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    /**
     * @notice Returns the ContractSet that represents the implementations for a given network.
     */
    function getImplementations() internal view returns (ContractSet memory) {
        ContractSet memory set = implementations[block.chainid];
        require(set.L1CrossDomainMessenger != address(0), "no implementations for this network");
        return set;
    }

    /**
     * @notice Returns the ContractSet that represents the proxies for a given network.
     */
    function getProxies() internal view returns (ContractSet memory) {
        ContractSet memory set = proxies[block.chainid];
        require(set.L1CrossDomainMessenger != address(0), "no proxies for this network");
        return set;
    }
}
