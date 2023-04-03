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
import { Predeploys } from "../../contracts/libraries/Predeploys.sol";
import { SystemConfig } from "../../contracts/L1/SystemConfig.sol";
import { ResourceMetering } from "../../contracts/L1/ResourceMetering.sol";
import { Semver } from "../../contracts/universal/Semver.sol";

/**
 * @title PostSherlockL2
 * @notice Upgrade script for upgrading the L2 predeploy implementations after the sherlock audit.
 *         Assumes that a gnosis safe is used as the privileged account and the same
 *         gnosis safe is the owner the proxy admin.
 *         This could be optimized by checking for the number of approvals up front
 *         and not submitting the final approval as `execTransaction` can be called when
 *         there are `threshold - 1` approvals.
 *         Uses the "approved hashes" method of interacting with the gnosis safe. Allows
 *         for the most simple user experience when using automation and no indexer.
 *         Run the command without the `--broadcast` flag and it will print a tenderly URL.
 */
contract PostSherlockL2 is Script {
    /**
     * @notice Interface for multicall3.
     */
    IMulticall3 private constant multicall = IMulticall3(MULTICALL3_ADDRESS);

    /**
     * @notice OP Mainnet chain id.
     */
    uint256 constant OP_MAINNET = 10;

    /**
     * @notice OP Goerli chain id.
     */
    uint256 constant OP_GOERLI = 420;

    /**
     * @notice The proxy admin predeploy on L2.
     */
    ProxyAdmin constant PROXY_ADMIN = ProxyAdmin(0x4200000000000000000000000000000000000018);

    /**
     * @notice Represents a set of L2 predepploy contracts. Used to represent a set of
     *         implementations and also a set of proxies.
     */
    struct ContractSet {
        address BaseFeeVault;
        address GasPriceOracle;
        address L1Block;
        address L1FeeVault;
        address L2CrossDomainMessenger;
        address L2ERC721Bridge;
        address L2StandardBridge;
        address L2ToL1MessagePasser;
        address SequencerFeeVault;
        address OptimismMintableERC20Factory;
        address OptimismMintableERC721Factory;
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
    string constant internal BaseFeeVault_Version = "1.1.0";
    string constant internal GasPriceOracle_Version = "1.0.0";
    string constant internal L1Block_Version = "1.0.0";
    string constant internal L1FeeVault_Version = "1.1.0";
    string constant internal L2CrossDomainMessenger_Version = "1.1.0";
    string constant internal L2ERC721Bridge_Version = "1.1.0";
    string constant internal L2StandardBridge_Version = "1.1.0";
    string constant internal L2ToL1MessagePasser_Version = "1.0.0";
    string constant internal SequencerFeeVault_Version = "1.1.0";
    string constant internal OptimismMintableERC20Factory_Version = "1.1.0";
    string constant internal OptimismMintableERC721Factory_Version = "1.1.0";

    /**
     * @notice Place the contract addresses in storage so they can be used when building calldata.
     */
    function setUp() external {
        implementations[OP_GOERLI] = ContractSet({
            BaseFeeVault: 0xEcBb01757B6b7799465a422aD0fC7Fd5F5179F0a,
            GasPriceOracle: 0x79f09f735B2d1a42fF864C014d3bD4aA5FAA6A5E,
            L1Block: 0xd5F2B9f6Ee80065b2Ce18bF1e629c5aC1C98c7F6,
            L1FeeVault: 0x9bA5E286934F0A29fb2f8421f60d3eE8A853447C,
            L2CrossDomainMessenger: 0xDe90fE30325588D895Ee4c2E862E703e165a01c7,
            L2ERC721Bridge: 0x777adA49d40DAC02AE5b4FdC292feDf9066435A3,
            L2StandardBridge: 0x3EA657c5aA0E4Bce1D8919dC7f248724d7B0987a,
            L2ToL1MessagePasser: 0xEF2ec5A5465f075E010BE70966a8667c94BCe15a,
            SequencerFeeVault: 0x4781674AAe242bbDf6C58b81Cf4F06F1534cd37d,
            OptimismMintableERC20Factory: 0xeDF90ac13642e6445955b79CdDA321ecB136b29B,
            OptimismMintableERC721Factory: 0x795F355F75f9B28AEC6cC6A887704191e630065b
        });

        proxies[OP_GOERLI] = ContractSet({
            BaseFeeVault: Predeploys.BASE_FEE_VAULT,
            GasPriceOracle: Predeploys.GAS_PRICE_ORACLE,
            L1Block: Predeploys.L1_BLOCK_ATTRIBUTES,
            L1FeeVault: Predeploys.L1_FEE_VAULT,
            L2CrossDomainMessenger: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            L2ERC721Bridge: Predeploys.L2_ERC721_BRIDGE,
            L2StandardBridge: Predeploys.L2_STANDARD_BRIDGE,
            L2ToL1MessagePasser: Predeploys.L2_TO_L1_MESSAGE_PASSER,
            SequencerFeeVault: Predeploys.SEQUENCER_FEE_WALLET,
            OptimismMintableERC20Factory: Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY,
            OptimismMintableERC721Factory: Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY
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
        require(_versionHash(prox.BaseFeeVault) == keccak256(bytes(BaseFeeVault_Version)));
        require(_versionHash(prox.GasPriceOracle) == keccak256(bytes(GasPriceOracle_Version)));
        require(_versionHash(prox.L1Block) == keccak256(bytes(L1Block_Version)));
        require(_versionHash(prox.L1FeeVault) == keccak256(bytes(L1FeeVault_Version)));
        require(_versionHash(prox.L2CrossDomainMessenger) == keccak256(bytes(L2CrossDomainMessenger_Version)));
        require(_versionHash(prox.L2ERC721Bridge) == keccak256(bytes(L2ERC721Bridge_Version)));
        require(_versionHash(prox.L2StandardBridge) == keccak256(bytes(L2StandardBridge_Version)));
        require(_versionHash(prox.L2ToL1MessagePasser) == keccak256(bytes(L2ToL1MessagePasser_Version)));
        require(_versionHash(prox.SequencerFeeVault) == keccak256(bytes(SequencerFeeVault_Version)));
        require(_versionHash(prox.OptimismMintableERC20Factory) == keccak256(bytes(OptimismMintableERC20Factory_Version)));
        require(_versionHash(prox.OptimismMintableERC721Factory) == keccak256(bytes(OptimismMintableERC721Factory_Version)));

        // Check that the codehashes of all implementations match the proxies set implementations.
        ContractSet memory impl = getImplementations();
        require(PROXY_ADMIN.getProxyImplementation(prox.BaseFeeVault).codehash == impl.BaseFeeVault.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.GasPriceOracle).codehash == impl.GasPriceOracle.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L1Block).codehash == impl.L1Block.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L1FeeVault).codehash == impl.L1FeeVault.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2CrossDomainMessenger).codehash == impl.L2CrossDomainMessenger.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2ERC721Bridge).codehash == impl.L2ERC721Bridge.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2StandardBridge).codehash == impl.L2StandardBridge.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.L2ToL1MessagePasser).codehash == impl.L2ToL1MessagePasser.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.SequencerFeeVault).codehash == impl.SequencerFeeVault.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.OptimismMintableERC20Factory).codehash == impl.OptimismMintableERC20Factory.codehash);
        require(PROXY_ADMIN.getProxyImplementation(prox.OptimismMintableERC721Factory).codehash == impl.OptimismMintableERC721Factory.codehash);
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

        if (block.chainid == OP_GOERLI) {
            safe = 0xE534ccA2753aCFbcDBCeB2291F596fc60495257e;
            proxyAdmin = 0x4200000000000000000000000000000000000018;
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
     *         A total of 9 calls are made to the proxy admin to upgrade the implementations
     *         of the predeploys.
     */
    function buildCalldata(address _proxyAdmin) internal view returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](11);

        ContractSet memory impl = getImplementations();
        ContractSet memory prox = getProxies();

        // Upgrade the BaseFeeVault
        calls[0] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.BaseFeeVault), impl.BaseFeeVault)
            )
        });

        // Upgrade the GasPriceOracle
        calls[1] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.GasPriceOracle), impl.GasPriceOracle)
            )
        });

        // Upgrade the L1Block predeploy
        calls[2] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1Block), impl.L1Block)
            )
        });

        // Upgrade the L1FeeVault
        calls[3] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L1FeeVault), impl.L1FeeVault)
            )
        });

        // Upgrade the L2CrossDomainMessenger
        calls[4] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2CrossDomainMessenger), impl.L2CrossDomainMessenger)
            )
        });

        // Upgrade the L2ERC721Bridge
        calls[5] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2ERC721Bridge), impl.L2ERC721Bridge)
            )
        });

        // Upgrade the L2StandardBridge
        calls[6] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2StandardBridge), impl.L2StandardBridge)
            )
        });

        // Upgrade the L2ToL1MessagePasser
        calls[7] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.L2ToL1MessagePasser), impl.L2ToL1MessagePasser)
            )
        });

        // Upgrade the SequencerFeeVault
        calls[8] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.SequencerFeeVault), impl.SequencerFeeVault)
            )
        });

        // Upgrade the OptimismMintableERC20Factory
        calls[9] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismMintableERC20Factory), impl.OptimismMintableERC20Factory)
            )
        });

        // Upgrade the OptimismMintableERC721Factory
        calls[10] = IMulticall3.Call3({
            target: _proxyAdmin,
            allowFailure: false,
            callData: abi.encodeCall(
                ProxyAdmin.upgrade,
                (payable(prox.OptimismMintableERC721Factory), impl.OptimismMintableERC721Factory)
            )
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    /**
     * @notice Returns the ContractSet that represents the implementations for a given network.
     */
    function getImplementations() internal view returns (ContractSet memory) {
        ContractSet memory set = implementations[block.chainid];
        require(set.BaseFeeVault != address(0), "no implementations for this network");
        return set;
    }

    /**
     * @notice Returns the ContractSet that represents the proxies for a given network.
     */
    function getProxies() internal view returns (ContractSet memory) {
        ContractSet memory set = proxies[block.chainid];
        require(set.BaseFeeVault != address(0), "no proxies for this network");
        return set;
    }
}
