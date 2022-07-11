//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { L1StandardBridge } from "../L1/L1StandardBridge.sol";
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { OptimismMintableERC20Factory } from "../universal/OptimismMintableERC20Factory.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { L2CrossDomainMessenger } from "../L2/L2CrossDomainMessenger.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { LegacyERC20ETH } from "../legacy/LegacyERC20ETH.sol";
import { PredeployAddresses } from "../libraries/PredeployAddresses.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { Proxy } from "../universal/Proxy.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ResolvedDelegateProxy } from "../legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "../legacy/AddressManager.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";
import { iL1ChugSplashDeployer } from "../legacy/L1ChugSplashProxy.sol";

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

    function _setUp() public {
        // Give alice and bob some ETH
        vm.deal(alice, 1 << 16);
        vm.deal(bob, 1 << 16);
        vm.deal(multisig, 1 << 16);

        vm.label(alice, "alice");
        vm.label(bob, "bob");
        vm.label(multisig, "multisig");

        // Make sure we have a non-zero base fee
        vm.fee(1000000000);
    }
}

contract L2OutputOracle_Initializer is CommonTest {
    // Test target
    L2OutputOracle oracle;
    L2OutputOracle oracleImpl;

    // Constructor arguments
    address sequencer = 0x000000000000000000000000000000000000AbBa;
    address owner = 0x000000000000000000000000000000000000ACDC;
    uint256 submissionInterval = 1800;
    uint256 l2BlockTime = 2;
    bytes32 genesisL2Output = keccak256(abi.encode(0));
    uint256 historicalTotalBlocks = 199;
    uint256 startingBlockNumber = 200;
    uint256 startingTimestamp = 1000;

    // Test data
    uint256 initL1Time;

    // Advance the evm's time to meet the L2OutputOracle's requirements for appendL2Output
    function warpToAppendTime(uint256 _nextBlockNumber) public {
        vm.warp(oracle.computeL2Timestamp(_nextBlockNumber) + 1);
    }

    function setUp() public virtual {
        _setUp();

        // By default the first block has timestamp and number zero, which will cause underflows in the
        // tests, so we'll move forward to these block values.
        initL1Time = startingTimestamp + 1;
        vm.warp(initL1Time);
        vm.roll(startingBlockNumber);
        // Deploy the L2OutputOracle and transfer owernship to the sequencer
        oracleImpl = new L2OutputOracle(
            submissionInterval,
            genesisL2Output,
            historicalTotalBlocks,
            startingBlockNumber,
            startingTimestamp,
            l2BlockTime,
            sequencer,
            owner
        );
        Proxy proxy = new Proxy(multisig);
        vm.prank(multisig);
        proxy.upgradeToAndCall(
            address(oracleImpl),
            abi.encodeWithSelector(
                L2OutputOracle.initialize.selector,
                genesisL2Output,
                startingBlockNumber,
                sequencer,
                owner
            )
        );
        oracle = L2OutputOracle(address(proxy));
    }
}

contract Portal_Initializer is L2OutputOracle_Initializer {
    // Test target
    OptimismPortal opImpl;
    OptimismPortal op;

    function setUp() public virtual override {
        L2OutputOracle_Initializer.setUp();
        opImpl = new OptimismPortal(oracle, 7 days);
        Proxy proxy = new Proxy(multisig);
        vm.prank(multisig);
        proxy.upgradeToAndCall(
            address(opImpl),
            abi.encodeWithSelector(OptimismPortal.initialize.selector)
        );
        op = OptimismPortal(payable(address(proxy)));
    }
}

contract Messenger_Initializer is L2OutputOracle_Initializer {
    OptimismPortal op;
    AddressManager addressManager;
    L1CrossDomainMessenger L1Messenger;
    L2CrossDomainMessenger L2Messenger =
        L2CrossDomainMessenger(PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER);
    L2ToL1MessagePasser messagePasser =
        L2ToL1MessagePasser(payable(PredeployAddresses.L2_TO_L1_MESSAGE_PASSER));

    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );

    event WithdrawalInitiated(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );

    event RelayedMessage(bytes32 indexed msgHash);

    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 mint,
        uint256 value,
        uint64 gasLimit,
        bool isCreation,
        bytes data
    );

    event WithdrawalFinalized(bytes32 indexed, bool success);

    function setUp() public virtual override {
        super.setUp();

        // Deploy the OptimismPortal
        op = new OptimismPortal(oracle, 7 days);
        vm.label(address(op), "OptimismPortal");

        // Deploy the address manager
        vm.prank(multisig);
        addressManager = new AddressManager();

        // Setup implementation
        L1CrossDomainMessenger L1MessengerImpl = new L1CrossDomainMessenger(op);

        // Setup the address manager and proxy
        vm.prank(multisig);
        addressManager.setAddress("OVM_L1CrossDomainMessenger", address(L1MessengerImpl));
        ResolvedDelegateProxy proxy = new ResolvedDelegateProxy(
            addressManager,
            "OVM_L1CrossDomainMessenger"
        );
        L1Messenger = L1CrossDomainMessenger(address(proxy));
        L1Messenger.initialize();

        vm.etch(
            PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER,
            address(new L2CrossDomainMessenger(address(L1Messenger))).code
        );

        L2Messenger.initialize(address(L1Messenger));

        // Set the L2ToL1MessagePasser at the correct address
        vm.etch(
            PredeployAddresses.L2_TO_L1_MESSAGE_PASSER,
            address(new L2ToL1MessagePasser()).code
        );

        // Label addresses
        vm.label(address(addressManager), "AddressManager");
        vm.label(address(L1MessengerImpl), "L1CrossDomainMessenger_Impl");
        vm.label(address(L1Messenger), "L1CrossDomainMessenger_Proxy");
        vm.label(PredeployAddresses.LEGACY_ERC20_ETH, "LegacyERC20ETH");
        vm.label(PredeployAddresses.L2_TO_L1_MESSAGE_PASSER, "L2ToL1MessagePasser");
        vm.label(PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER, "L2CrossDomainMessenger");

        vm.label(
            AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)),
            "L1CrossDomainMessenger_aliased"
        );
    }
}

contract Bridge_Initializer is Messenger_Initializer {
    L1StandardBridge L1Bridge;
    L2StandardBridge L2Bridge;
    OptimismMintableERC20Factory L2TokenFactory;
    OptimismMintableERC20Factory L1TokenFactory;
    ERC20 L1Token;
    OptimismMintableERC20 L2Token;
    ERC20 NativeL2Token;
    OptimismMintableERC20 RemoteL1Token;

    event ETHDepositInitiated(
        address indexed from,
        address indexed to,
        uint256 amount,
        bytes data
    );

    event ETHWithdrawalFinalized(
        address indexed from,
        address indexed to,
        uint256 amount,
        bytes data
    );

    event ERC20DepositInitiated(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event ERC20WithdrawalFinalized(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event WithdrawalInitiated(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event DepositFinalized(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event DepositFailed(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event ETHBridgeInitiated(
        address indexed from,
        address indexed to,
        uint256 amount,
        bytes data
    );

    event ETHBridgeFinalized(
        address indexed from,
        address indexed to,
        uint256 amount,
        bytes data
    );

    event ERC20BridgeInitiated(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event ERC20BridgeFinalized(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    event ERC20BridgeFailed(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 amount,
        bytes data
    );

    function setUp() public virtual override {
        super.setUp();

        vm.label(PredeployAddresses.L2_STANDARD_BRIDGE, "L2StandardBridge");
        vm.label(PredeployAddresses.OPTIMISM_MINTABLE_ERC20_FACTORY, "OptimismMintableERC20Factory");

        // Deploy the L1 bridge and initialize it with the address of the
        // L1CrossDomainMessenger
        L1ChugSplashProxy proxy = new L1ChugSplashProxy(multisig);
        vm.mockCall(
            multisig,
            abi.encodeWithSelector(iL1ChugSplashDeployer.isUpgrading.selector),
            abi.encode(true)
        );
        vm.startPrank(multisig);
        proxy.setCode(address(new L1StandardBridge(payable(address(L1Messenger)))).code);
        vm.clearMockedCalls();
        address L1Bridge_Impl = proxy.getImplementation();
        vm.stopPrank();

        L1Bridge = L1StandardBridge(payable(address(proxy)));
        L1Bridge.initialize(payable(address(L1Messenger)));

        vm.label(address(proxy), "L1StandardBridge_Proxy");
        vm.label(address(L1Bridge_Impl), "L1StandardBridge_Impl");

        // Deploy the L2StandardBridge, move it to the correct predeploy
        // address and then initialize it
        L2StandardBridge l2B = new L2StandardBridge(payable(PredeployAddresses.L2_STANDARD_BRIDGE));
        vm.etch(PredeployAddresses.L2_STANDARD_BRIDGE, address(l2B).code);
        L2Bridge = L2StandardBridge(payable(PredeployAddresses.L2_STANDARD_BRIDGE));
        L2Bridge.initialize(payable(address(L1Bridge)));

        // Set up the L2 mintable token factory
        OptimismMintableERC20Factory factory = new OptimismMintableERC20Factory(
            PredeployAddresses.L2_STANDARD_BRIDGE
        );
        vm.etch(PredeployAddresses.OPTIMISM_MINTABLE_ERC20_FACTORY, address(factory).code);
        L2TokenFactory = OptimismMintableERC20Factory(
            PredeployAddresses.OPTIMISM_MINTABLE_ERC20_FACTORY
        );

        vm.etch(PredeployAddresses.LEGACY_ERC20_ETH, address(new LegacyERC20ETH()).code);

        L1Token = new ERC20("Native L1 Token", "L1T");

        // Deploy the L2 ERC20 now
        L2Token = OptimismMintableERC20(
            L2TokenFactory.createStandardL2Token(
                address(L1Token),
                string(abi.encodePacked("L2-", L1Token.name())),
                string(abi.encodePacked("L2-", L1Token.symbol()))
            )
        );

        NativeL2Token = new ERC20("Native L2 Token", "L2T");
        L1TokenFactory = new OptimismMintableERC20Factory(address(L1Bridge));

        RemoteL1Token = OptimismMintableERC20(
            L1TokenFactory.createStandardL2Token(
                address(NativeL2Token),
                string(abi.encodePacked("L1-", NativeL2Token.name())),
                string(abi.encodePacked("L1-", NativeL2Token.symbol()))
            )
        );
    }
}

// Used for testing a future upgrade beyond the current implementations.
// We include some variables so that we can sanity check accessing storage values after an upgrade.
contract NextImpl is Initializable {
    // Initializable occupies the zero-th slot.
    bytes32 slot1;
    bytes32[19] __gap;
    bytes32 slot21;
    bytes32 public constant slot21Init = bytes32(hex"1337");

    function initialize() public reinitializer(2) {
        // Slot21 is unused by an of our upgradeable contracts.
        // This is used to verify that we can access this value after an upgrade.
        slot21 = slot21Init;
    }
}
