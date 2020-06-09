pragma solidity ^0.5.0;

/**
 * @title StubSafetyChecker
 * @notice This stubbed safety checker always returns TRUE when `isBytecodeSafe(...) is called.
 */
contract StubSafetyChecker {
    /**
     * @notice Returns true.
     * @param _bytecode The bytecode to safety check. This will be ignored.
     */
    function isBytecodeSafe(
        bytes memory _bytecode
    ) public view returns (bool) {
        return true;
    }
}
