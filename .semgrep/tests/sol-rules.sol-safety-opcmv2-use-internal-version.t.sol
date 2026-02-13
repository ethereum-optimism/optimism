contract SemgrepTest__sol_safety_opcmv2_use_internal_version {
    function badUsage() public view returns (string memory) {
        // ruleid: sol-safety-opcmv2-use-internal-version
        return version();
    }

    // ok: sol-safety-opcmv2-use-internal-version
    function version() public pure returns (string memory) {
        return "1.0.0";
    }

    function _version() internal view returns (string memory) {
        // ok: sol-safety-opcmv2-use-internal-version
        return thisOPCM.version();
    }

    function goodUsage() public view returns (string memory) {
        // ok: sol-safety-opcmv2-use-internal-version
        return _version();
    }

    function externalCall() public view returns (string memory) {
        // ok: sol-safety-opcmv2-use-internal-version
        return someContract.version();
    }
}
