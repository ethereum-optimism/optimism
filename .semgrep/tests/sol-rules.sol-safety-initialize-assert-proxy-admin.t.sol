// Semgrep tests for Solidity rules are defined in this file.
// Semgrep tests do not need to be valid Solidity code but should be syntactically correct so that
// Semgrep can parse them. You don't need to be able to *run* the code here but it should look like
// the code that you expect to catch with the rule.
//
// Semgrep testing 101
// Use comments like "ruleid: <rule-id>" to assert that the rule catches the code.
// Use comments like "ok: <rule-id>" to assert that the rule does not catch the code.

contract SemgrepTest__sol_safety_initialize_assert_proxy_admin {
    // ok: sol-safety-initialize-assert-proxy-admin
    function initialize(address _param) external reinitializer(1) {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        _setParam(_param);
    }

    // ok: sol-safety-initialize-assert-proxy-admin
    function initialize() external initializer {
        _assertOnlyProxyAdminOrProxyAdminOwner();
    }

    // ok: sol-safety-initialize-assert-proxy-admin
    function initialize(address _a, uint256 _b) public reinitializer(2) {
        // Comment before assertion
        _assertOnlyProxyAdminOrProxyAdminOwner();
        // More initialization
        param = _a;
    }

    // ruleid: sol-safety-initialize-assert-proxy-admin
    function initialize(address _param) external reinitializer(1) {
        _setParam(_param);
    }

    // ruleid: sol-safety-initialize-assert-proxy-admin
    function initialize() external initializer {
        __Ownable_init();
    }

    // ruleid: sol-safety-initialize-assert-proxy-admin
    function initialize(address _a, uint256 _b) public initializer {
        param = _a;
        value = _b;
    }
}
