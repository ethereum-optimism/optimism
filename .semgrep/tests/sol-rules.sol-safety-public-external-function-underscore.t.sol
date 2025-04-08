// Semgrep tests for Solidity rules are defined in this file.
// Semgrep tests do not need to be valid Solidity code but should be syntactically correct so that
// Semgrep can parse them. You don't need to be able to *run* the code here but it should look like
// the code that you expect to catch with the rule.
//
// Semgrep testing 101
// Use comments like "ruleid: <rule-id>" to assert that the rule catches the code.
// Use comments like "ok: <rule-id>" to assert that the rule does not catch the code.

/// NOTE: Semgrep limitations mean that the rule for this check is defined as a relatively loose regex that searches the
/// remainder of the file after the `@custom:proxied` natspec tag is detected. This means that we must test the case
/// without this natspec tag BEFORE the case with the tag or the rule will apply to the remainder of the file.

contract SemgrepTest__sol_safety_public_external_function_underscore {
    // ruleid: sol-safety-public-external-function-underscore
    function _invalidPublicFunction() public {
        // ...
    }

    // ruleid: sol-safety-public-external-function-underscore
    function _invalidExternalFunction() external {
        // ...
    }

    // ruleid: sol-safety-public-external-function-underscore
    function _invalidPublicWithParams(uint256 _num) public returns (bool) {
        // ...
    }

    // ruleid: sol-safety-public-external-function-underscore
    function _invalidExternalWithParams(uint256 _num) external returns (bool) {
        // ...
    }

    // ruleid: sol-safety-public-external-function-underscore
    function _invalidPublicWithModifiers() public view returns (uint256) {
        // ...
    }

    // ruleid: sol-safety-public-external-function-underscore
    function _invalidExternalWithModifiers() external view returns (uint256) {
        // ...
    }

    // ok: sol-safety-public-external-function-underscore
    function validPublicFunction() public {
        // ...
    }

    // ok: sol-safety-public-external-function-underscore
    function validExternalFunction() external {
        // ...
    }

    // ok: sol-safety-public-external-function-underscore
    function validPublicWithParams(uint256 _num) public pure returns (bool) {
        // ...
    }

    // ok: sol-safety-public-external-function-underscore
    function validExternalWithParams(uint256 _num) external pure returns (bool) {
        // ...
    }

    // ok: sol-safety-public-external-function-underscore
    function publicFunction() public {
        // ...
    }

    // ok: sol-safety-public-external-function-underscore
    function externalFunction() external {
        // ...
    }
}
