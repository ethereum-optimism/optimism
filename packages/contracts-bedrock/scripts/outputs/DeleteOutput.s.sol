// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";

import { LibSort } from "../libraries/LibSort.sol";
import { IGnosisSafe, Enum } from "../interfaces/IGnosisSafe.sol";
import { SafeBuilder } from "../universal/SafeBuilder.sol";

import { Types } from "../../contracts/libraries/Types.sol";
import { FeeVault } from "../../contracts/universal/FeeVault.sol";
import { L2OutputOracle } from "../../contracts/L1/L2OutputOracle.sol";
import { Predeploys } from "../../contracts/libraries/Predeploys.sol";

/**
 * @title DeleteOutput
 * @notice Deletes an output root from the L2OutputOracle.
 * @notice Example usage is provided in the README documentation.
 */
contract DeleteOutput is SafeBuilder {
    /**
     * @notice A set of contract addresses for the script.
     */
    struct ContractSet {
        address Safe;
        address ProxyAdmin;
        address L2OutputOracleProxy;
    }

    /**
     * @notice A mapping of chainid to a ContractSet.
     */
    mapping(uint256 => ContractSet) internal _contracts;

    /**
     * @notice The l2 output index we will delete.
     */
    uint256 internal index;

    /**
     * @notice The address of the L2OutputOracle to target.
     */
    address internal oracle;

    /**
     * @notice Place the contract addresses in storage for ux.
     */
    function setUp() external {
        _contracts[GOERLI] = ContractSet({
            Safe: 0xBc1233d0C3e6B5d53Ab455cF65A6623F6dCd7e4f,
            ProxyAdmin: 0x01d3670863c3F4b24D7b107900f0b75d4BbC6e0d,
            L2OutputOracleProxy: 0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0
        });
    }

    /**
     * @notice Returns the ContractSet for the defined block chainid.
     *
     * @dev Reverts if no ContractSet is defined.
     */
    function contracts() public view returns (ContractSet memory) {
        ContractSet memory cs = _contracts[block.chainid];
        if (cs.Safe == address(0) || cs.ProxyAdmin == address(0) || cs.L2OutputOracleProxy == address(0)) {
            revert("Missing Contract Set for the given block.chainid");
        }
        return cs;
    }

    /**
     * @notice Executes the gnosis safe transaction to delete an L2 Output Root.
     */
    function run(uint256 _index) external returns (bool) {
        address _safe = contracts().Safe;
        address _proxyAdmin = contracts().ProxyAdmin;
        index = _index;
        return run(_safe, _proxyAdmin);
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal view override {
        L2OutputOracle l2oo = L2OutputOracle(contracts().L2OutputOracleProxy);
        Types.OutputProposal memory proposal = l2oo.getL2Output(index);
        require(proposal.l2BlockNumber == 0, "DeleteOutput: Output deletion failed.");
    }

    /**
     * @notice Test coverage of the script.
     */
    function test_script_succeeds() skipWhenNotForking external {
        uint256 _index = getLatestIndex();
        require(_index != 0, "DeleteOutput: No outputs to delete.");

        index = _index;

        address safe = contracts().Safe;
        require(safe != address(0), "DeleteOutput: Invalid safe address.");

        address proxyAdmin = contracts().ProxyAdmin;
        require(proxyAdmin != address(0), "DeleteOutput: Invalid proxy admin address.");

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

    function buildCalldata(address _proxyAdmin) internal view override returns (bytes memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](1);

        calls[0] = IMulticall3.Call3({
            target: oracle,
            allowFailure: false,
            callData: abi.encodeCall(
                L2OutputOracle.deleteL2Outputs,
                (index)
            )
        });

        return abi.encodeCall(IMulticall3.aggregate3, (calls));
    }

    /**
     * @notice Computes the safe transaction hash.
     */
    function computeSafeTransactionHash(uint256 _index) public returns (bytes32) {
        ContractSet memory cs = contracts();
        address _safe = cs.Safe;
        address _proxyAdmin = cs.ProxyAdmin;
        index = _index;
        oracle = cs.L2OutputOracleProxy;

        return _getTransactionHash(_safe, _proxyAdmin);
    }

    /**
     * @notice Returns the challenger for the L2OutputOracle.
     */
    function getChallenger() public view returns (address) {
        L2OutputOracle l2oo = L2OutputOracle(contracts().L2OutputOracleProxy);
        return l2oo.CHALLENGER();
    }

    /**
     * @notice Returns the L2 Block Number for the given index.
     */
    function getL2BlockNumber(uint256 _index) public view returns (uint256) {
        L2OutputOracle l2oo = L2OutputOracle(contracts().L2OutputOracleProxy);
        return l2oo.getL2Output(_index).l2BlockNumber;
    }

    /**
     * @notice Returns the output root for the given index.
     */
    function getOutputFromIndex(uint256 _index) public view returns (bytes32) {
        L2OutputOracle l2oo = L2OutputOracle(contracts().L2OutputOracleProxy);
        return l2oo.getL2Output(_index).outputRoot;
    }

    /**
     * @notice Returns the output root with the corresponding to the L2 Block Number.
     */
    function getOutputFromL2BlockNumber(uint256 l2BlockNumber) public view returns (bytes32) {
        L2OutputOracle l2oo = L2OutputOracle(contracts().L2OutputOracleProxy);
        return l2oo.getL2OutputAfter(l2BlockNumber).outputRoot;
    }

    /**
     * @notice Returns the latest l2 output index.
     */
    function getLatestIndex() public view returns (uint256) {
        L2OutputOracle l2oo = L2OutputOracle(contracts().L2OutputOracleProxy);
        return l2oo.latestOutputIndex();
    }
}
