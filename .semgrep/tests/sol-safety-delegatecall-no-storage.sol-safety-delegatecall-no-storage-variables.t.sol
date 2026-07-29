contract SemgrepTest__sol_safety_delegatecall_no_storage_variables {
    // ok: sol-safety-delegatecall-no-storage-variables
    address internal immutable THIS_L2CM;

    // ok: sol-safety-delegatecall-no-storage-variables
    bytes32 internal constant INITIALIZABLE_SLOT_OZ_V4 = bytes32(0);

    // ok: sol-safety-delegatecall-no-storage-variables
    string public constant version = "1.1.0";

    // ok: sol-safety-delegatecall-no-storage-variables
    address internal immutable STORAGE_SETTER_IMPL;

    // ruleid: sol-safety-delegatecall-no-storage-variables
    uint256 public storageVar;

    // ruleid: sol-safety-delegatecall-no-storage-variables
    address internal someAddr;

    // ruleid: sol-safety-delegatecall-no-storage-variables
    mapping(address => uint256) internal balances;

    // ruleid: sol-safety-delegatecall-no-storage-variables
    bool private flag;

    // ruleid: sol-safety-delegatecall-no-storage-variables
    bytes32 public someHash;

    // ruleid: sol-safety-delegatecall-no-storage-variables
    string public name;
}
