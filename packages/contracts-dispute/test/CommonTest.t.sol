// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, StdUtils } from "forge-std/Test.sol";

contract CommonTest is Test {
    address alice = address(128);
    address bob = address(256);
    address multisig = address(512);

    address immutable ZERO_ADDRESS = address(0);
    address immutable NON_ZERO_ADDRESS = address(1);
    uint256 immutable NON_ZERO_VALUE = 100;
    uint256 immutable ZERO_VALUE = 0;
    uint64 immutable NON_ZERO_GASLIMIT = 50000;
    bytes32 nonZeroHash = keccak256(abi.encode("NON_ZERO"));
    bytes NON_ZERO_DATA = hex"0000111122223333444455556666777788889999aaaabbbbccccddddeeeeffff0000";

    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 indexed version,
        bytes opaqueData
    );

    FFIInterface ffi;

    function setUp() public virtual {
        // Give alice and bob some ETH
        vm.deal(alice, 1 << 16);
        vm.deal(bob, 1 << 16);
        vm.deal(multisig, 1 << 16);

        vm.label(alice, "alice");
        vm.label(bob, "bob");
        vm.label(multisig, "multisig");

        // Make sure we have a non-zero base fee
        vm.fee(1000000000);

        ffi = new FFIInterface();
    }

    function emitTransactionDeposited(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    ) internal {
        emit TransactionDeposited(
            _from,
            _to,
            0,
            abi.encodePacked(_mint, _value, _gasLimit, _isCreation, _data)
        );
    }
}

contract L2OutputOracle_Initializer is CommonTest {
    L2OutputOracle oracle;
    L2OutputOracle oracleImpl;
    BondManager bondManager;
    DisputeGameFactory disputeGameFactory;

    L2ToL1MessagePasser messagePasser =
        L2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER));

    // Constructor arguments
    address internal owner = 0x000000000000000000000000000000000000ACDC;
    uint256 internal l2BlockTime = 2;
    uint256 internal startingBlockNumber = 200;
    uint256 internal startingTimestamp = 0;
    uint256 internal minimumProposalCost = 1 ether;
    uint256 internal finalizationPeriodSeconds = 7 days;
    address guardian;

    // Test data
    uint256 initL1Time;

    event OutputProposed(
        bytes32 indexed outputRoot,
        uint256 indexed l2OutputIndex,
        uint256 indexed l2BlockNumber,
        uint256 l1Timestamp
    );

    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    // Advance the evm's time to meet the L2OutputOracle's requirements for proposeL2Output
    function warpToProposeTime(uint256 _nextBlockNumber) public {
        vm.warp(oracle.computeL2Timestamp(_nextBlockNumber) + 1);
    }

    function createMockAttestationGame() public returns (address) {
        Claim rootClaim = Claim.wrap(bytes32(""));
        bytes memory extraData = bytes("");
        MockAttestationDisputeGame implementation = new MockAttestationDisputeGame();
        GameType gt = GameType.ATTESTATION;
        disputeGameFactory.setImplementation(gt, IDisputeGame(address(implementation)));

        address proxy = address(disputeGameFactory.create(gt, rootClaim, extraData));

        return proxy;
    }

    function createEmptyAttestationGame() public returns (address) {
        MockAttestationDisputeGame emptyGame = new MockAttestationDisputeGame();
        return address(emptyGame);
    }

    function setUp() public virtual override {
        super.setUp();
        guardian = makeAddr("guardian");

        disputeGameFactory = new DisputeGameFactory(address(guardian));
        bondManager = new BondManager(disputeGameFactory);

        // By default the first block has timestamp and number zero, which will cause underflows in the
        // tests, so we'll move forward to these block values.
        initL1Time = startingTimestamp + 1;
        vm.warp(initL1Time);
        vm.roll(startingBlockNumber);
        // Deploy the L2OutputOracle and transfer owernship to the proposer
        oracleImpl = new L2OutputOracle({
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: startingTimestamp,
            _finalizationPeriodSeconds: finalizationPeriodSeconds,
            _bondManager: IBondManager(address(bondManager)),
            _disputeGameFactory: IDisputeGameFactory(address(disputeGameFactory))
        });
        Proxy proxy = new Proxy(multisig);
        vm.prank(multisig);
        proxy.upgradeToAndCall(
            address(oracleImpl),
            abi.encodeCall(L2OutputOracle.initialize, (startingBlockNumber, startingTimestamp))
        );
        oracle = L2OutputOracle(address(proxy));
        vm.label(address(oracle), "L2OutputOracle");

        // Set the L2ToL1MessagePasser at the correct address
        vm.etch(Predeploys.L2_TO_L1_MESSAGE_PASSER, address(new L2ToL1MessagePasser()).code);

        vm.label(Predeploys.L2_TO_L1_MESSAGE_PASSER, "L2ToL1MessagePasser");
    }
}


