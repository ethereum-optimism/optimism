pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title OwnershipPredicate
 * @notice TODO
 */
contract OwnershipPredicate {

    /*** Structs ***/
    struct Range {
        uint256 start;
        uint256 end;
    }

    struct StateObject {
        address predicateAddress;
        bytes data;
    }

    struct StateUpdate {
        Range range;
        StateObject stateObject;
        address plasmaContract;
        uint256 plasmaBlockNumber;
    }

    struct OwnershipTransaction {
        UnsignedOwnershipTransaction unsignedTransaction;
        bytes ecdsaSignature;
    }

    struct UnsignedOwnershipTransaction {
        address plasmaContract;
        Range range;
        bytes32 methodId;
        Parameters parameters;
    }

    struct Parameters {
        StateObject newState;
        uint64 originBlock;
        uint64 maxBlock;
    }

    /*** Events ***/
    event TestEncoding(
        bytes encoding
    );
    event TestEncoding2(
        StateObject stateObject
    );
    event TestEncoding3(
        StateUpdate stateUpdate
    );

    function testEncoding() public returns (bool) {
        // Test encoding with a state object
        StateObject memory obj = StateObject(0xE620c291f4f6706313BD7d8CB1775C50b4A97e16, '0x1234');
        bytes memory test = abi.encode(obj);
        emit TestEncoding(test);
        StateObject memory test2 = abi.decode(test, (StateObject));
        emit TestEncoding2(test2);
        bytes memory test2Encoding = abi.encode(obj);
        emit TestEncoding(test2Encoding);

        // Next test encoding with a state update
        Range memory range = Range(0x100, 0x200);
        StateUpdate memory stateUpdate = StateUpdate(range, obj, 0xE620c291f4f6706313BD7d8CB1775C50b4A97e16, 0x05);
        emit TestEncoding3(stateUpdate);
        bytes memory stateUpdateEncoding = abi.encode(stateUpdate);
        emit TestEncoding(stateUpdateEncoding);
        StateUpdate memory stateUpdate2 = abi.decode(stateUpdateEncoding, (StateUpdate));
        emit TestEncoding3(stateUpdate2);

        // Retrun true for no reason
        return true;
    }

    function verifyStateTransition(StateUpdate memory preState, OwnershipTransaction memory transaction, StateUpdate memory postState) public returns (bool) {
        // TODO: Actually verify everything
        return true;
    }
}
