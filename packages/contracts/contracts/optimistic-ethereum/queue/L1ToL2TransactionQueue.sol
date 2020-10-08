pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { CanonicalTransactionChain } from "../chain/CanonicalTransactionChain.sol";
import { RollupQueue } from "./RollupQueue.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { GasConsumer } from "../utils/libraries/GasConsumer.sol";

/**
 * @title L1ToL2TransactionQueue
 */
contract L1ToL2TransactionQueue is ContractResolver, RollupQueue {
    /*
     * Events
     */

    event L1ToL2TxEnqueued(
        address sender,
        address target,
        uint32 gasLimit,
        bytes data
    );

    /*
     * Constants
     */

    uint constant public L2_GAS_DISCOUNT_DIVISOR = 10;

    address public l1MessengerAddress;

    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     */
    constructor(
        address _addressResolver
    )
        public
        ContractResolver(_addressResolver)
    {
    }

    /*
     * Public Functions
     */

    function tempInit(address _l1MessengerAddress) public {
        require(l1MessengerAddress == address(0));
        l1MessengerAddress = _l1MessengerAddress;
    }

    /**
     * Checks that that a dequeue is authenticated, and dequques if authenticated.
     */
    function dequeue()
        public
    {
        require(msg.sender == address(resolveCanonicalTransactionChain()), "Only the canonical transaction chain can dequeue L1->L2 queue transactions.");
        _dequeue();
    }

    /**
     * Enqueues an L1->L2 message.
     */
    function enqueueL1ToL2Message(
        address _ovmTarget,
        uint32 _ovmGasLimit,
        bytes calldata _data
    )
        external
    {
        require(l1MessengerAddress == address(0) || msg.sender == l1MessengerAddress);

        uint gasToBurn = _ovmGasLimit / L2_GAS_DISCOUNT_DIVISOR;
        resolveGasConsumer().consumeGasInternalCall(gasToBurn);

        address sender = msg.sender;
        bytes memory tx = encodeL1ToL2Tx(
            sender,
            _ovmTarget,
            _ovmGasLimit,
            _data
        );
        emit L1ToL2TxEnqueued(
            sender,
            _ovmTarget,
            _ovmGasLimit,
            _data
        );
        _enqueue(tx);
    }

    /*
     * Internal Functions
     */

    function encodeL1ToL2Tx(
        address _sender,
        address _target,
        uint32 _gasLimit,
        bytes memory _data
    ) 
        internal
        returns(bytes memory)
    {
        return abi.encode(_sender, _target, _gasLimit, _data);
    }

    /*
     * Contract Resolution
     */

    function resolveCanonicalTransactionChain()
        internal
        view
        returns (CanonicalTransactionChain)
    {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }

    function resolveGasConsumer()
        internal
        view
        returns (GasConsumer)
    {
        return GasConsumer(resolveContract("GasConsumer"));
    }
}
