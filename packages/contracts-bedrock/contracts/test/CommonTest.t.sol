// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test, StdUtils } from "forge-std/Test.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { L1StandardBridge } from "../L1/L1StandardBridge.sol";
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { L1ERC721Bridge } from "../L1/L1ERC721Bridge.sol";
import { L2ERC721Bridge } from "../L2/L2ERC721Bridge.sol";
import { OptimismMintableERC20Factory } from "../universal/OptimismMintableERC20Factory.sol";
import { OptimismMintableERC721Factory } from "../universal/OptimismMintableERC721Factory.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { L2CrossDomainMessenger } from "../L2/L2CrossDomainMessenger.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { LegacyERC20ETH } from "../legacy/LegacyERC20ETH.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { Types } from "../libraries/Types.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { Proxy } from "../universal/Proxy.sol";
import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import { ResolvedDelegateProxy } from "../legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "../legacy/AddressManager.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";
import { IL1ChugSplashDeployer } from "../legacy/L1ChugSplashProxy.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { IAllowanceTransfer } from "../libraries/Permit2Lib.sol";

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
    // Test target
    L2OutputOracle oracle;
    L2OutputOracle oracleImpl;

    L2ToL1MessagePasser messagePasser =
        L2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER));

    // Constructor arguments
    address internal proposer = 0x000000000000000000000000000000000000AbBa;
    address internal owner = 0x000000000000000000000000000000000000ACDC;
    uint256 internal submissionInterval = 1800;
    uint256 internal l2BlockTime = 2;
    uint256 internal startingBlockNumber = 200;
    uint256 internal startingTimestamp = 1000;

    // Test data
    uint256 initL1Time;

    // Advance the evm's time to meet the L2OutputOracle's requirements for proposeL2Output
    function warpToProposeTime(uint256 _nextBlockNumber) public {
        vm.warp(oracle.computeL2Timestamp(_nextBlockNumber) + 1);
    }

    function setUp() public virtual {
        _setUp();

        // By default the first block has timestamp and number zero, which will cause underflows in the
        // tests, so we'll move forward to these block values.
        initL1Time = startingTimestamp + 1;
        vm.warp(initL1Time);
        vm.roll(startingBlockNumber);
        // Deploy the L2OutputOracle and transfer owernship to the proposer
        oracleImpl = new L2OutputOracle(
            submissionInterval,
            l2BlockTime,
            startingBlockNumber,
            startingTimestamp,
            proposer,
            owner
        );
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
        L2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );

    event SentMessageExtension1(address indexed sender, uint256 value);

    event MessagePassed(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data,
        bytes32 withdrawalHash
    );

    event RelayedMessage(bytes32 indexed msgHash);
    event FailedRelayedMessage(bytes32 indexed msgHash);

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

    event WhatHappened(bool success, bytes returndata);

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
        L1Messenger.initialize(alice);

        vm.etch(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            address(new L2CrossDomainMessenger(address(L1Messenger))).code
        );

        L2Messenger.initialize();

        // Label addresses
        vm.label(address(addressManager), "AddressManager");
        vm.label(address(L1MessengerImpl), "L1CrossDomainMessenger_Impl");
        vm.label(address(L1Messenger), "L1CrossDomainMessenger_Proxy");
        vm.label(Predeploys.LEGACY_ERC20_ETH, "LegacyERC20ETH");
        vm.label(Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L2CrossDomainMessenger");

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
    ERC20 BadL1Token;
    OptimismMintableERC20 L2Token;
    ERC20 NativeL2Token;
    ERC20 BadL2Token;
    OptimismMintableERC20 RemoteL1Token;

    IAllowanceTransfer public constant PERMIT2 =
        IAllowanceTransfer(address(0x000000000022D473030F116dDEE9F6B43aC78BA3));

    event ETHDepositInitiated(address indexed from, address indexed to, uint256 amount, bytes data);

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

    event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes data);

    event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes data);

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

    function setUp() public virtual override {
        super.setUp();

        Permit2Deployer deployer = new Permit2Deployer();
        deployer.deploy();
        vm.label(deployer.PERMIT2(), "permit2");

        vm.label(Predeploys.L2_STANDARD_BRIDGE, "L2StandardBridge");
        vm.label(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY, "OptimismMintableERC20Factory");

        // Deploy the L1 bridge and initialize it with the address of the
        // L1CrossDomainMessenger
        L1ChugSplashProxy proxy = new L1ChugSplashProxy(multisig);
        vm.mockCall(
            multisig,
            abi.encodeWithSelector(IL1ChugSplashDeployer.isUpgrading.selector),
            abi.encode(true)
        );
        vm.startPrank(multisig);
        proxy.setCode(address(new L1StandardBridge(payable(address(L1Messenger)))).code);
        vm.clearMockedCalls();
        address L1Bridge_Impl = proxy.getImplementation();
        vm.stopPrank();

        L1Bridge = L1StandardBridge(payable(address(proxy)));

        vm.label(address(proxy), "L1StandardBridge_Proxy");
        vm.label(address(L1Bridge_Impl), "L1StandardBridge_Impl");

        // Deploy the L2StandardBridge, move it to the correct predeploy
        // address and then initialize it
        L2StandardBridge l2B = new L2StandardBridge(payable(proxy));
        vm.etch(Predeploys.L2_STANDARD_BRIDGE, address(l2B).code);
        L2Bridge = L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE));

        // Set up the L2 mintable token factory
        OptimismMintableERC20Factory factory = new OptimismMintableERC20Factory(
            Predeploys.L2_STANDARD_BRIDGE
        );
        vm.etch(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY, address(factory).code);
        L2TokenFactory = OptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);

        vm.etch(Predeploys.LEGACY_ERC20_ETH, address(new LegacyERC20ETH()).code);

        L1Token = new ERC20("Native L1 Token", "L1T");

        // Deploy the L2 ERC20 now
        L2Token = OptimismMintableERC20(
            L2TokenFactory.createStandardL2Token(
                address(L1Token),
                string(abi.encodePacked("L2-", L1Token.name())),
                string(abi.encodePacked("L2-", L1Token.symbol()))
            )
        );

        BadL2Token = OptimismMintableERC20(
            L2TokenFactory.createStandardL2Token(
                address(1),
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

        BadL1Token = OptimismMintableERC20(
            L1TokenFactory.createStandardL2Token(
                address(1),
                string(abi.encodePacked("L1-", NativeL2Token.name())),
                string(abi.encodePacked("L1-", NativeL2Token.symbol()))
            )
        );
    }
}

contract ERC721Bridge_Initializer is Messenger_Initializer {
    L1ERC721Bridge L1Bridge;
    L2ERC721Bridge L2Bridge;

    function setUp() public virtual override {
        super.setUp();

        // Deploy the L1ERC721Bridge.
        L1Bridge = new L1ERC721Bridge(address(L1Messenger), Predeploys.L2_ERC721_BRIDGE);

        // Deploy the implementation for the L2ERC721Bridge and etch it into the predeploy address.
        vm.etch(
            Predeploys.L2_ERC721_BRIDGE,
            address(new L2ERC721Bridge(Predeploys.L2_CROSS_DOMAIN_MESSENGER, address(L1Bridge)))
                .code
        );

        // Set up a reference to the L2ERC721Bridge.
        L2Bridge = L2ERC721Bridge(Predeploys.L2_ERC721_BRIDGE);

        // Label the L1 and L2 bridges.
        vm.label(address(L1Bridge), "L1ERC721Bridge");
        vm.label(address(L2Bridge), "L2ERC721Bridge");
    }
}

contract FFIInterface is Test {
    function getProveWithdrawalTransactionInputs(Types.WithdrawalTransaction memory _tx)
        external
        returns (
            bytes32,
            bytes32,
            bytes32,
            bytes32,
            bytes[] memory
        )
    {
        string[] memory cmds = new string[](9);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "getProveWithdrawalTransactionInputs";
        cmds[3] = vm.toString(_tx.nonce);
        cmds[4] = vm.toString(_tx.sender);
        cmds[5] = vm.toString(_tx.target);
        cmds[6] = vm.toString(_tx.value);
        cmds[7] = vm.toString(_tx.gasLimit);
        cmds[8] = vm.toString(_tx.data);

        bytes memory result = vm.ffi(cmds);
        (
            bytes32 stateRoot,
            bytes32 storageRoot,
            bytes32 outputRoot,
            bytes32 withdrawalHash,
            bytes[] memory withdrawalProof
        ) = abi.decode(result, (bytes32, bytes32, bytes32, bytes32, bytes[]));

        return (stateRoot, storageRoot, outputRoot, withdrawalHash, withdrawalProof);
    }

    function hashCrossDomainMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external returns (bytes32) {
        string[] memory cmds = new string[](9);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "hashCrossDomainMessage";
        cmds[3] = vm.toString(_nonce);
        cmds[4] = vm.toString(_sender);
        cmds[5] = vm.toString(_target);
        cmds[6] = vm.toString(_value);
        cmds[7] = vm.toString(_gasLimit);
        cmds[8] = vm.toString(_data);

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (bytes32));
    }

    function hashWithdrawal(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external returns (bytes32) {
        string[] memory cmds = new string[](9);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "hashWithdrawal";
        cmds[3] = vm.toString(_nonce);
        cmds[4] = vm.toString(_sender);
        cmds[5] = vm.toString(_target);
        cmds[6] = vm.toString(_value);
        cmds[7] = vm.toString(_gasLimit);
        cmds[8] = vm.toString(_data);

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (bytes32));
    }

    function hashOutputRootProof(
        bytes32 _version,
        bytes32 _stateRoot,
        bytes32 _messagePasserStorageRoot,
        bytes32 _latestBlockhash
    ) external returns (bytes32) {
        string[] memory cmds = new string[](7);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "hashOutputRootProof";
        cmds[3] = Strings.toHexString(uint256(_version));
        cmds[4] = Strings.toHexString(uint256(_stateRoot));
        cmds[5] = Strings.toHexString(uint256(_messagePasserStorageRoot));
        cmds[6] = Strings.toHexString(uint256(_latestBlockhash));

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (bytes32));
    }

    function hashDepositTransaction(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gas,
        bytes memory _data,
        uint256 _logIndex
    ) external returns (bytes32) {
        string[] memory cmds = new string[](11);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "hashDepositTransaction";
        cmds[3] = "0x0000000000000000000000000000000000000000000000000000000000000000";
        cmds[4] = vm.toString(_logIndex);
        cmds[5] = vm.toString(_from);
        cmds[6] = vm.toString(_to);
        cmds[7] = vm.toString(_mint);
        cmds[8] = vm.toString(_value);
        cmds[9] = vm.toString(_gas);
        cmds[10] = vm.toString(_data);

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (bytes32));
    }

    function encodeDepositTransaction(Types.UserDepositTransaction calldata txn)
        external
        returns (bytes memory)
    {
        string[] memory cmds = new string[](12);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "encodeDepositTransaction";
        cmds[3] = vm.toString(txn.from);
        cmds[4] = vm.toString(txn.to);
        cmds[5] = vm.toString(txn.value);
        cmds[6] = vm.toString(txn.mint);
        cmds[7] = vm.toString(txn.gasLimit);
        cmds[8] = vm.toString(txn.isCreation);
        cmds[9] = vm.toString(txn.data);
        cmds[10] = vm.toString(txn.l1BlockHash);
        cmds[11] = vm.toString(txn.logIndex);

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (bytes));
    }

    function encodeCrossDomainMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external returns (bytes memory) {
        string[] memory cmds = new string[](9);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "encodeCrossDomainMessage";
        cmds[3] = vm.toString(_nonce);
        cmds[4] = vm.toString(_sender);
        cmds[5] = vm.toString(_target);
        cmds[6] = vm.toString(_value);
        cmds[7] = vm.toString(_gasLimit);
        cmds[8] = vm.toString(_data);

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (bytes));
    }

    function decodeVersionedNonce(uint256 nonce) external returns (uint256, uint256) {
        string[] memory cmds = new string[](4);
        cmds[0] = "node";
        cmds[1] = "dist/scripts/differential-testing.js";
        cmds[2] = "decodeVersionedNonce";
        cmds[3] = vm.toString(nonce);

        bytes memory result = vm.ffi(cmds);
        return abi.decode(result, (uint256, uint256));
    }

    function getMerkleTrieFuzzCase(string memory variant)
        external
        returns (
            bytes32,
            bytes memory,
            bytes memory,
            bytes[] memory
        )
    {
        string[] memory cmds = new string[](5);
        cmds[0] = "./test-case-generator/fuzz";
        cmds[1] = "-m";
        cmds[2] = "trie";
        cmds[3] = "-v";
        cmds[4] = variant;

        return abi.decode(vm.ffi(cmds), (bytes32, bytes, bytes, bytes[]));
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

contract Reverter {
    fallback() external {
        revert();
    }
}

// Useful for testing reentrancy guards
contract CallerCaller {
    event WhatHappened(bool success, bytes returndata);

    fallback() external {
        (bool success, bytes memory returndata) = msg.sender.call(msg.data);
        emit WhatHappened(success, returndata);
        assembly {
            switch success
            case 0 {
                revert(add(returndata, 0x20), mload(returndata))
            }
            default {
                return(add(returndata, 0x20), mload(returndata))
            }
        }
    }
}

contract Permit2Deployer is Test {
    address constant public PERMIT2 = 0x000000000022D473030F116dDEE9F6B43aC78BA3;

    function deploy() external {
        bytes memory bytecode = hex"6040608081526004908136101561001557600080fd5b600090813560e01c80630d58b1db1461126c578063137c29fe146110755780632a2d80d114610db75780632b67b57014610bde57806330f28b7a14610ade5780633644e51514610a9d57806336c7851614610a285780633ff9dcb1146109a85780634fe02b441461093f57806365d9723c146107ac57806387517c451461067a578063927da105146105c3578063cc53287f146104a3578063edd9444b1461033a5763fe8ec1a7146100c657600080fd5b346103365760c07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126103365767ffffffffffffffff833581811161033257610114903690860161164b565b60243582811161032e5761012b903690870161161a565b6101336114e6565b9160843585811161032a5761014b9036908a016115c1565b98909560a43590811161032657610164913691016115c1565b969095815190610173826113ff565b606b82527f5065726d697442617463685769746e6573735472616e7366657246726f6d285460208301527f6f6b656e5065726d697373696f6e735b5d207065726d69747465642c61646472838301527f657373207370656e6465722c75696e74323536206e6f6e63652c75696e74323560608301527f3620646561646c696e652c000000000000000000000000000000000000000000608083015282519a8b9181610222602085018096611f93565b918237018a8152039961025b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe09b8c8101835282611437565b5190209085515161026b81611ebb565b908a5b8181106102f95750506102f6999a6102ed9183516102a081610294602082018095611f66565b03848101835282611437565b519020602089810151858b015195519182019687526040820192909252336060820152608081019190915260a081019390935260643560c08401528260e081015b03908101835282611437565b51902093611cf7565b80f35b8061031161030b610321938c5161175e565b51612054565b61031b828661175e565b52611f0a565b61026e565b8880fd5b8780fd5b8480fd5b8380fd5b5080fd5b5091346103365760807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126103365767ffffffffffffffff9080358281116103325761038b903690830161164b565b60243583811161032e576103a2903690840161161a565b9390926103ad6114e6565b9160643590811161049f576103c4913691016115c1565b949093835151976103d489611ebb565b98885b81811061047d5750506102f697988151610425816103f9602082018095611f66565b037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08101835282611437565b5190206020860151828701519083519260208401947ffcf35f5ac6a2c28868dc44c302166470266239195f02b0ee408334829333b7668652840152336060840152608083015260a082015260a081526102ed8161141b565b808b61031b8261049461030b61049a968d5161175e565b9261175e565b6103d7565b8680fd5b5082346105bf57602090817ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126103325780359067ffffffffffffffff821161032e576104f49136910161161a565b929091845b848110610504578580f35b8061051a610515600193888861196c565b61197c565b61052f84610529848a8a61196c565b0161197c565b3389528385528589209173ffffffffffffffffffffffffffffffffffffffff80911692838b528652868a20911690818a5285528589207fffffffffffffffffffffffff000000000000000000000000000000000000000081541690558551918252848201527f89b1add15eff56b3dfe299ad94e01f2b52fbcb80ae1a3baea6ae8c04cb2b98a4853392a2016104f9565b8280fd5b50346103365760607ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033657610676816105ff6114a0565b936106086114c3565b6106106114e6565b73ffffffffffffffffffffffffffffffffffffffff968716835260016020908152848420928816845291825283832090871683528152919020549251938316845260a083901c65ffffffffffff169084015260d09190911c604083015281906060820190565b0390f35b50346103365760807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc360112610336576106b26114a0565b906106bb6114c3565b916106c46114e6565b65ffffffffffff926064358481169081810361032a5779ffffffffffff0000000000000000000000000000000000000000947fda9fa7c1b00402c17d0161b249b1ab8bbec047c5a52207b9c112deffd817036b94338a5260016020527fffffffffffff0000000000000000000000000000000000000000000000000000858b209873ffffffffffffffffffffffffffffffffffffffff809416998a8d5260205283878d209b169a8b8d52602052868c209486156000146107a457504216925b8454921697889360a01b16911617179055815193845260208401523392a480f35b905092610783565b5082346105bf5760607ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf576107e56114a0565b906107ee6114c3565b9265ffffffffffff604435818116939084810361032a57338852602091600183528489209673ffffffffffffffffffffffffffffffffffffffff80911697888b528452858a20981697888a5283528489205460d01c93848711156109175761ffff9085840316116108f05750907f55eb90d810e1700b35a8e7e25395ff7f2b2259abd7415ca2284dfb1c246418f393929133895260018252838920878a528252838920888a5282528389209079ffffffffffffffffffffffffffffffffffffffffffffffffffff7fffffffffffff000000000000000000000000000000000000000000000000000083549260d01b16911617905582519485528401523392a480f35b84517f24d35a26000000000000000000000000000000000000000000000000000000008152fd5b5084517f756688fe000000000000000000000000000000000000000000000000000000008152fd5b503461033657807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc360112610336578060209273ffffffffffffffffffffffffffffffffffffffff61098f6114a0565b1681528084528181206024358252845220549051908152f35b5082346105bf57817ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf577f3704902f963766a4e561bbaab6e6cdc1b1dd12f6e9e99648da8843b3f46b918d90359160243533855284602052818520848652602052818520818154179055815193845260208401523392a280f35b8234610a9a5760807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc360112610a9a57610a606114a0565b610a686114c3565b610a706114e6565b6064359173ffffffffffffffffffffffffffffffffffffffff8316830361032e576102f6936117a1565b80fd5b503461033657817ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033657602090610ad7611b1e565b9051908152f35b508290346105bf576101007ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf57610b1a3661152a565b90807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7c36011261033257610b4c611478565b9160e43567ffffffffffffffff8111610bda576102f694610b6f913691016115c1565b939092610b7c8351612054565b6020840151828501519083519260208401947f939c21a48a8dbe3a9a2404a1d46691e4d39f6583d6ec6b35714604c986d801068652840152336060840152608083015260a082015260a08152610bd18161141b565b51902091611c25565b8580fd5b509134610336576101007ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033657610c186114a0565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdc360160c08112610332576080855191610c51836113e3565b1261033257845190610c6282611398565b73ffffffffffffffffffffffffffffffffffffffff91602435838116810361049f578152604435838116810361049f57602082015265ffffffffffff606435818116810361032a5788830152608435908116810361049f576060820152815260a435938285168503610bda576020820194855260c4359087830182815260e43567ffffffffffffffff811161032657610cfe90369084016115c1565b929093804211610d88575050918591610d786102f6999a610d7e95610d238851611fbe565b90898c511690519083519260208401947ff3841cd1ff0085026a6327b620b67997ce40f282c88a8e905a7a5626e310f3d086528401526060830152608082015260808152610d70816113ff565b519020611bd9565b916120c7565b519251169161199d565b602492508a51917fcd21db4f000000000000000000000000000000000000000000000000000000008352820152fd5b5091346103365760607ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc93818536011261033257610df36114a0565b9260249081359267ffffffffffffffff9788851161032a578590853603011261049f578051978589018981108282111761104a578252848301358181116103265785019036602383011215610326578382013591610e50836115ef565b90610e5d85519283611437565b838252602093878584019160071b83010191368311611046578801905b828210610fe9575050508a526044610e93868801611509565b96838c01978852013594838b0191868352604435908111610fe557610ebb90369087016115c1565b959096804211610fba575050508998995151610ed681611ebb565b908b5b818110610f9757505092889492610d7892610f6497958351610f02816103f98682018095611f66565b5190209073ffffffffffffffffffffffffffffffffffffffff9a8b8b51169151928551948501957faf1b0d30d2cab0380e68f0689007e3254993c596f2fdd0aaa7f4d04f794408638752850152830152608082015260808152610d70816113ff565b51169082515192845b848110610f78578580f35b80610f918585610f8b600195875161175e565b5161199d565b01610f6d565b80610311610fac8e9f9e93610fb2945161175e565b51611fbe565b9b9a9b610ed9565b8551917fcd21db4f000000000000000000000000000000000000000000000000000000008352820152fd5b8a80fd5b6080823603126110465785608091885161100281611398565b61100b85611509565b8152611018838601611509565b838201526110278a8601611607565b8a8201528d611037818701611607565b90820152815201910190610e7a565b8c80fd5b84896041867f4e487b7100000000000000000000000000000000000000000000000000000000835252fd5b5082346105bf576101407ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf576110b03661152a565b91807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7c360112610332576110e2611478565b67ffffffffffffffff93906101043585811161049f5761110590369086016115c1565b90936101243596871161032a57611125610bd1966102f6983691016115c1565b969095825190611134826113ff565b606482527f5065726d69745769746e6573735472616e7366657246726f6d28546f6b656e5060208301527f65726d697373696f6e73207065726d69747465642c6164647265737320737065848301527f6e6465722c75696e74323536206e6f6e63652c75696e7432353620646561646c60608301527f696e652c0000000000000000000000000000000000000000000000000000000060808301528351948591816111e3602085018096611f93565b918237018b8152039361121c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe095868101835282611437565b5190209261122a8651612054565b6020878101518589015195519182019687526040820192909252336060820152608081019190915260a081019390935260e43560c08401528260e081016102e1565b5082346105bf576020807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033257813567ffffffffffffffff92838211610bda5736602383011215610bda5781013592831161032e576024906007368386831b8401011161049f57865b8581106112e5578780f35b80821b83019060807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdc83360301126103265761139288876001946060835161132c81611398565b611368608461133c8d8601611509565b9485845261134c60448201611509565b809785015261135d60648201611509565b809885015201611509565b918291015273ffffffffffffffffffffffffffffffffffffffff80808093169516931691166117a1565b016112da565b6080810190811067ffffffffffffffff8211176113b457604052565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6060810190811067ffffffffffffffff8211176113b457604052565b60a0810190811067ffffffffffffffff8211176113b457604052565b60c0810190811067ffffffffffffffff8211176113b457604052565b90601f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0910116810190811067ffffffffffffffff8211176113b457604052565b60c4359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b600080fd5b6004359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b6024359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b6044359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc01906080821261149b576040805190611563826113e3565b8082941261149b57805181810181811067ffffffffffffffff8211176113b457825260043573ffffffffffffffffffffffffffffffffffffffff8116810361149b578152602435602082015282526044356020830152606435910152565b9181601f8401121561149b5782359167ffffffffffffffff831161149b576020838186019501011161149b57565b67ffffffffffffffff81116113b45760051b60200190565b359065ffffffffffff8216820361149b57565b9181601f8401121561149b5782359167ffffffffffffffff831161149b576020808501948460061b01011161149b57565b91909160608184031261149b576040805191611666836113e3565b8294813567ffffffffffffffff9081811161149b57830182601f8201121561149b578035611693816115ef565b926116a087519485611437565b818452602094858086019360061b8501019381851161149b579086899897969594939201925b8484106116e3575050505050855280820135908501520135910152565b90919293949596978483031261149b578851908982019082821085831117611730578a928992845261171487611509565b81528287013583820152815201930191908897969594936116c6565b602460007f4e487b710000000000000000000000000000000000000000000000000000000081526041600452fd5b80518210156117725760209160051b010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b92919273ffffffffffffffffffffffffffffffffffffffff604060008284168152600160205282828220961695868252602052818120338252602052209485549565ffffffffffff8760a01c16804211611884575082871696838803611812575b5050611810955016926118b5565b565b878484161160001461184f57602488604051907ff96fb0710000000000000000000000000000000000000000000000000000000082526004820152fd5b7fffffffffffffffffffffffff000000000000000000000000000000000000000084846118109a031691161790553880611802565b602490604051907fd81b2f2e0000000000000000000000000000000000000000000000000000000082526004820152fd5b9060006064926020958295604051947f23b872dd0000000000000000000000000000000000000000000000000000000086526004860152602485015260448401525af13d15601f3d116001600051141617161561190e57565b60646040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f5452414e534645525f46524f4d5f4641494c45440000000000000000000000006044820152fd5b91908110156117725760061b0190565b3573ffffffffffffffffffffffffffffffffffffffff8116810361149b5790565b9065ffffffffffff908160608401511673ffffffffffffffffffffffffffffffffffffffff908185511694826020820151169280866040809401511695169560009187835260016020528383208984526020528383209916988983526020528282209184835460d01c03611af5579185611ace94927fc6a377bfc4eb120024a8ac08eef205be16b817020812c73223e81d1bdb9708ec98979694508715600014611ad35779ffffffffffff00000000000000000000000000000000000000009042165b60a01b167fffffffffffff00000000000000000000000000000000000000000000000000006001860160d01b1617179055519384938491604091949373ffffffffffffffffffffffffffffffffffffffff606085019616845265ffffffffffff809216602085015216910152565b0390a4565b5079ffffffffffff000000000000000000000000000000000000000087611a60565b600484517f756688fe000000000000000000000000000000000000000000000000000000008152fd5b467f000000000000000000000000000000000000000000000000000000000000000503611b69577f2094d195a0a2f55df37ecdac57ba3ccecf1180f3bd83392e42d48c91720f4f0390565b60405160208101907f8cad95687ba82c2ce50e74f7b754645e5117c3a5bec8151c0726d5857980a86682527f9ac997416e8ff9d2ff6bebeb7149f65cdae5e32e2b90440b566bb3044041d36a604082015246606082015230608082015260808152611bd3816113ff565b51902090565b611be1611b1e565b906040519060208201927f190100000000000000000000000000000000000000000000000000000000000084526022830152604282015260428152611bd381611398565b9192909360a435936040840151804211611cc65750602084510151808611611c955750918591610d78611c6594611c60602088015186611e47565b611bd9565b73ffffffffffffffffffffffffffffffffffffffff809151511692608435918216820361149b57611810936118b5565b602490604051907f3728b83d0000000000000000000000000000000000000000000000000000000082526004820152fd5b602490604051907fcd21db4f0000000000000000000000000000000000000000000000000000000082526004820152fd5b959093958051519560409283830151804211611e175750848803611dee57611d2e918691610d7860209b611c608d88015186611e47565b60005b868110611d42575050505050505050565b611d4d81835161175e565b5188611d5a83878a61196c565b01359089810151808311611dbe575091818888886001968596611d84575b50505050505001611d31565b611db395611dad9273ffffffffffffffffffffffffffffffffffffffff6105159351169561196c565b916118b5565b803888888883611d78565b6024908651907f3728b83d0000000000000000000000000000000000000000000000000000000082526004820152fd5b600484517fff633a38000000000000000000000000000000000000000000000000000000008152fd5b6024908551907fcd21db4f0000000000000000000000000000000000000000000000000000000082526004820152fd5b9073ffffffffffffffffffffffffffffffffffffffff600160ff83161b9216600052600060205260406000209060081c6000526020526040600020818154188091551615611e9157565b60046040517f756688fe000000000000000000000000000000000000000000000000000000008152fd5b90611ec5826115ef565b611ed26040519182611437565b8281527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0611f0082946115ef565b0190602036910137565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114611f375760010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b805160208092019160005b828110611f7f575050505090565b835185529381019392810192600101611f71565b9081519160005b838110611fab575050016000815290565b8060208092840101518185015201611f9a565b60405160208101917f65626cad6cb96493bf6f5ebea28756c966f023ab9e8a83a7101849d5573b3678835273ffffffffffffffffffffffffffffffffffffffff8082511660408401526020820151166060830152606065ffffffffffff9182604082015116608085015201511660a082015260a0815260c0810181811067ffffffffffffffff8211176113b45760405251902090565b6040516020808201927f618358ac3db8dc274f0cd8829da7e234bd48cd73c4a740aede1adec9846d06a1845273ffffffffffffffffffffffffffffffffffffffff81511660408401520151606082015260608152611bd381611398565b919082604091031261149b576020823592013590565b6000843b61222e5750604182036121ac576120e4828201826120b1565b939092604010156117725760209360009360ff6040608095013560f81c5b60405194855216868401526040830152606082015282805260015afa156121a05773ffffffffffffffffffffffffffffffffffffffff806000511691821561217657160361214c57565b60046040517f815e1d64000000000000000000000000000000000000000000000000000000008152fd5b60046040517f8baa579f000000000000000000000000000000000000000000000000000000008152fd5b6040513d6000823e3d90fd5b60408203612204576121c0918101906120b1565b91601b7f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff84169360ff1c019060ff8211611f375760209360009360ff608094612102565b60046040517f4be6321b000000000000000000000000000000000000000000000000000000008152fd5b929391601f928173ffffffffffffffffffffffffffffffffffffffff60646020957fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0604051988997889687947f1626ba7e000000000000000000000000000000000000000000000000000000009e8f8752600487015260406024870152816044870152868601378b85828601015201168101030192165afa9081156123a857829161232a575b507fffffffff000000000000000000000000000000000000000000000000000000009150160361230057565b60046040517fb0669cbc000000000000000000000000000000000000000000000000000000008152fd5b90506020813d82116123a0575b8161234460209383611437565b810103126103365751907fffffffff0000000000000000000000000000000000000000000000000000000082168203610a9a57507fffffffff0000000000000000000000000000000000000000000000000000000090386122d4565b3d9150612337565b6040513d84823e3d90fdfea164736f6c6343000811000a";
        vm.etch(PERMIT2, bytecode);
    }
}
