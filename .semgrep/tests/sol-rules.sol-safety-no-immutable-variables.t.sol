contract SemgrepTest__sol_safety_no_immutable_variables {
    // ok: sol-safety-no-immutable-variables
    uint256 public normalVar;

    // ok: sol-safety-no-immutable-variables
    address public normalAddress;

    // ok: sol-safety-no-immutable-variables
    string public normalString;

    // ok: sol-safety-no-immutable-variables
    bool public normalBool;

    // ruleid: sol-safety-no-immutable-variables
    uint256 immutable invalidImmutable1;

    // ruleid: sol-safety-no-immutable-variables
    bytes32 immutable invalidImmutable3 = "test";

    // ok: sol-safety-no-immutable-variables
    uint256 constant constantVar = 1;
}