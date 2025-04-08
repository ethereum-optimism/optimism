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

contract SemgrepTest__sol_safety_private_internal_function_underscore {
    // ok: sol-safety-private-internal-function-underscore
    function _validPrivateFunction() private {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _validInternalFunction() internal {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _validPrivateWithParams(uint256 _num) private returns (bool) {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _validInternalWithParams(uint256 _num) internal returns (bool) {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _validPrivateWithModifiers() private view returns (uint256) {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _validInternalWithModifiers() internal view returns (uint256) {
        // ...
    }

    // ruleid: sol-safety-private-internal-function-underscore
    function invalidPrivateFunction() private {
        // ...
    }

    // ruleid: sol-safety-private-internal-function-underscore
    function invalidInternalFunction() internal {
        // ...
    }

    // ruleid: sol-safety-private-internal-function-underscore
    function invalidPrivateWithParams(uint256 _num) private pure returns (bool) {
        // ...
    }

    // ruleid: sol-safety-private-internal-function-underscore
    function invalidInternalWithParams(uint256 _num) internal pure returns (bool) {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function publicFunction() public {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function externalFunction() external {
        // ...
    }
}

library SemgrepTestLibrary {
    // ok: sol-safety-private-internal-function-underscore
    function privateLibraryFunction() private {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function internalLibraryFunction() internal {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _privateLibraryFunctionWithUnderscore() private {
        // ...
    }

    // ok: sol-safety-private-internal-function-underscore
    function _internalLibraryFunctionWithUnderscore() internal {
        // ...
    }
}