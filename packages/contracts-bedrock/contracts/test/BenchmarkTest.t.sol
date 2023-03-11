// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import "./CommonTest.t.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { ResourceMetering } from "../L1/ResourceMetering.sol";

// Free function for setting the prevBaseFee param in the OptimismPortal.
function setPrevBaseFee(
    Vm _vm,
    address _op,
    uint128 _prevBaseFee
) {
    _vm.store(address(_op), bytes32(uint256(1)), bytes32((block.number << 192) | _prevBaseFee));
}

contract SetPrevBaseFee_Test is Portal_Initializer {
    function test_setPrevBaseFee_succeeds() external {
        setPrevBaseFee(vm, address(op), 100 gwei);
        (uint128 prevBaseFee, , uint64 prevBlockNum) = op.params();
        assertEq(uint256(prevBaseFee), 100 gwei);
        assertEq(uint256(prevBlockNum), block.number);
    }
}

// Tests for obtaining pure gas cost estimates for commonly used functions.
// The objective with these benchmarks is to strip down the actual test functions
// so that they are nothing more than the call we want measure the gas cost of.
// In order to achieve this we make no assertions, and handle everything else in the setUp()
// function.
contract GasBenchMark_OptimismPortal is Portal_Initializer {
    uint128 internal INITIAL_BASE_FEE;

    // Reusable default values for a test withdrawal
    Types.WithdrawalTransaction _defaultTx;

    uint256 _proposedOutputIndex;
    uint256 _proposedBlockNumber;
    bytes[] _withdrawalProof;
    Types.OutputRootProof internal _outputRootProof;
    bytes32 _outputRoot;

    // Use a constructor to set the storage vars above, so as to minimize the number of ffi calls.
    constructor() {
        super.setUp();
        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 100,
            gasLimit: 100_000,
            data: hex""
        });

        // Get withdrawal proof data we can use for testing.
        bytes32 _storageRoot;
        bytes32 _stateRoot;
        (_stateRoot, _storageRoot, _outputRoot, , _withdrawalProof) = ffi
            .getProveWithdrawalTransactionInputs(_defaultTx);

        // Setup a dummy output root proof for reuse.
        _outputRootProof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: _stateRoot,
            messagePasserStorageRoot: _storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });
        _proposedBlockNumber = oracle.nextBlockNumber();
        _proposedOutputIndex = oracle.nextOutputIndex();
    }

    // Get the system into a nice ready-to-use state.
    function setUp() public virtual override {
        // Configure the oracle to return the output root we've prepared.
        vm.warp(oracle.computeL2Timestamp(_proposedBlockNumber) + 1);
        vm.prank(oracle.PROPOSER());
        oracle.proposeL2Output(_outputRoot, _proposedBlockNumber, 0, 0);

        // Warp beyond the finalization period for the block we've proposed.
        vm.warp(
            oracle.getL2Output(_proposedOutputIndex).timestamp +
                oracle.FINALIZATION_PERIOD_SECONDS() +
                1
        );

        INITIAL_BASE_FEE = op.INITIAL_BASE_FEE();

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(op), 0xFFFFFFFF);
    }

    function test_depositTransaction_benchmark() external {
        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    function test_depositTransaction_benchmark_1() external {
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    function test_proveWithdrawalTransaction_benchmark() external {
        op.proveWithdrawalTransaction(
            _defaultTx,
            _proposedOutputIndex,
            _outputRootProof,
            _withdrawalProof
        );
    }
}

contract GasBenchMark_L1CrossDomainMessenger is Messenger_Initializer {
    uint128 internal INITIAL_BASE_FEE;

    function setUp() public virtual override {
        super.setUp();
        INITIAL_BASE_FEE = op.INITIAL_BASE_FEE();
    }

    function test_sendMessage_benchmark_0() external {
        vm.pauseGasMetering();
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        // The amount of data typically sent during a bridge deposit.
        bytes
            memory data = hex"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
        vm.resumeGasMetering();
        L1Messenger.sendMessage(bob, data, uint32(100));
    }

    function test_sendMessage_benchmark_1() external {
        vm.pauseGasMetering();
        setPrevBaseFee(vm, address(op), 10 gwei);
        // The amount of data typically sent during a bridge deposit.
        bytes
            memory data = hex"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
        vm.resumeGasMetering();
        L1Messenger.sendMessage(bob, data, uint32(100));
    }
}

contract GasBenchMark_L1StandardBridge_Deposit is Bridge_Initializer {
    uint128 internal INITIAL_BASE_FEE;

    function setUp() public virtual override {
        super.setUp();
        INITIAL_BASE_FEE = op.INITIAL_BASE_FEE();
        deal(address(L1Token), alice, 100000, true);
        vm.startPrank(alice, alice);
        L1Token.approve(address(L1Bridge), type(uint256).max);
    }

    function test_depositETH_benchmark_0() external {
        vm.pauseGasMetering();
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        vm.resumeGasMetering();
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }

    function test_depositETH_benchmark_1() external {
        vm.pauseGasMetering();
        setPrevBaseFee(vm, address(op), 10 gwei);
        vm.resumeGasMetering();
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }

    function test_depositERC20_benchmark_0() external {
        vm.pauseGasMetering();
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        vm.resumeGasMetering();
        L1Bridge.bridgeERC20({
            _localToken: address(L1Token),
            _remoteToken: address(L2Token),
            _amount: 100,
            _minGasLimit: 100_000,
            _extraData: hex""
        });
    }

    function test_depositERC20_benchmark_1() external {
        vm.pauseGasMetering();
        setPrevBaseFee(vm, address(op), 10 gwei);
        vm.resumeGasMetering();
        L1Bridge.bridgeERC20({
            _localToken: address(L1Token),
            _remoteToken: address(L2Token),
            _amount: 100,
            _minGasLimit: 100_000,
            _extraData: hex""
        });
    }
}

contract GasBenchMark_L1StandardBridge_Finalize is Bridge_Initializer {
    function setUp() public virtual override {
        super.setUp();
        deal(address(L1Token), address(L1Bridge), 100, true);
        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        vm.startPrank(address(L1Bridge.messenger()));
        vm.deal(address(L1Bridge.messenger()), 100);
    }

    function test_finalizeETHWithdrawal_benchmark() external {
        // TODO: Make this more accurate. It is underestimating the cost because it pranks
        // the call coming from the messenger, which bypasses the portal
        // and oracle.
        L1Bridge.finalizeETHWithdrawal{ value: 100 }(alice, alice, 100, hex"");
    }
}

contract GasBenchMark_L2OutputOracle is L2OutputOracle_Initializer {
    uint256 nextBlockNumber;

    function setUp() public override {
        super.setUp();
        nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.startPrank(proposer);
    }

    function test_proposeL2Output_benchmark() external {
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }
}
