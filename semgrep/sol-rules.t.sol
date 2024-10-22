// Semgrep tests for Solidity rules are defined in this file.
// Semgrep tests do not need to be valid Solidity code but should be syntactically correct so that
// Semgrep can parse them. You don't need to be able to *run* the code here but it should look like
// the code that you expect to catch with the rule.
//
// Semgrep testing 101
// Use comments like "ruleid: <rule-id>" to assert that the rule catches the code.
// Use comments like "ok: <rule-id>" to assert that the rule does not catch the code.

contract SemgrepTest__sol_safety_deployutils_args {
    function test() {
        // ruleid: sol-safety-deployutils-args
        DeployUtils.create1AndSave({
            _save: this,
            _name: "SuperchainConfig",
            _args: abi.encodeCall(ISuperchainConfig.__constructor__, ())
        });

        // ruleid: sol-safety-deployutils-args
        DeployUtils.create1({ _name: "SuperchainConfig", _args: abi.encodeCall(ISuperchainConfig.__constructor__, ()) });

        // ruleid: sol-safety-deployutils-args
        DeployUtils.create2AndSave({
            _save: this,
            _salt: _implSalt(),
            _name: "SuperchainConfig",
            _args: abi.encodeCall(ISuperchainConfig.__constructor__, ())
        });

        // ruleid: sol-safety-deployutils-args
        DeployUtils.create2({
            _salt: _implSalt(),
            _name: "SuperchainConfig",
            _args: abi.encodeCall(ISuperchainConfig.__constructor__, ())
        });

        // ok: sol-safety-deployutils-args
        DeployUtils.create1AndSave({
            _save: this,
            _name: "Proxy",
            _nick: "DataAvailabilityChallengeProxy",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (proxyAdmin)))
        });

        // ok: sol-safety-deployutils-args
        DeployUtils.create1({
            _name: "Proxy",
            _nick: "DataAvailabilityChallengeProxy",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (proxyAdmin)))
        });

        // ok: sol-safety-deployutils-args
        DeployUtils.create2AndSave({
            _save: this,
            _salt: _implSalt(),
            _name: "Proxy",
            _nick: "DataAvailabilityChallengeProxy",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (proxyAdmin)))
        });

        // ok: sol-safety-deployutils-args
        DeployUtils.create2({
            _salt: _implSalt(),
            _name: "Proxy",
            _nick: "DataAvailabilityChallengeProxy",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (proxyAdmin)))
        });
    }
}

contract SemgrepTest__sol_safety_expectrevert_before_ll_call {
    function test() {
        // ok: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        (bool revertsAsExpected,) = target.call(hex"");
        assertTrue(revertsAsExpected);

        // ok: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        (bool revertsAsExpected,) = target.delegatecall(hex"");
        assertTrue(revertsAsExpected);

        // ok: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        target.someFunction();

        // ruleid: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        (bool success,) = target.call(hex"");

        // ruleid: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        (bool success,) = target.call(hex"");
        assertTrue(success);

        // ruleid: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        (bool success,) = target.delegatecall(hex"");
        assertTrue(success);

        // ruleid: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        target.call(hex"");

        // ruleid: sol-safety-expectrevert-before-ll-call
        vm.expectRevert("some revert");
        target.delegatecall(hex"");
    }
}

contract SemgrepTest__sol_safety_expectrevert_no_args {
    function test() {
        // ok: sol-safety-expectrevert-no-args
        vm.expectRevert("some revert");
        target.someFunction();

        // ruleid: sol-safety-expectrevert-no-args
        vm.expectRevert();
        target.someFunction();
    }
}

contract SemgrepTest__sol_style_input_arg_fmt {
    // ok: sol-style-input-arg-fmt
    event Test(address indexed src, address indexed guy, uint256 wad);

    // ok: sol-style-input-arg-fmt
    function test() public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(address payable) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(uint256 _a, uint256 _b) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(uint256 _a, uint256 _b) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(bytes memory _a) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(bytes memory _a, uint256 _b) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(Contract.Struct memory _a) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(uint256 _b, bytes memory) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(bytes memory, uint256 _b) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test(bytes memory) public {
        // ...
    }

    // ok: sol-style-input-arg-fmt
    function test() public returns (bytes memory b_) {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(uint256 a) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(uint256 a, uint256 b) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(bytes memory a) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function testg(bytes memory a, uint256 b) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(uint256 b, bytes memory a) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(Contract.Struct memory a) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(uint256 _a, uint256 b) public {
        // ...
    }

    // ruleid: sol-style-input-arg-fmt
    function test(uint256 a, uint256 _b) public {
        // ...
    }
}

contract SemgrepTest__sol_style_return_arg_fmt {
    // ok: sol-style-return-arg-fmt
    function test() returns (uint256 a_) {
        // ...
    }

    // ok: sol-style-return-arg-fmt
    function test() returns (address payable) {
        // ...
    }

    // ok: sol-style-return-arg-fmt
    function test() returns (uint256 a_, bytes memory b_) {
        // ...
    }

    // ok: sol-style-return-arg-fmt
    function test() returns (Contract.Struct memory ab_) {
        // ...
    }

    // ok: sol-style-return-arg-fmt
    function test() returns (uint256, bool) {
        // ...
    }

    // ok: sol-style-return-arg-fmt
    function test() returns (uint256) {
        // ...
    }

    // ruleid: sol-style-return-arg-fmt
    function test() returns (uint256 a) {
        // ...
    }

    // ruleid: sol-style-return-arg-fmt
    function test() returns (uint256 a, bytes memory b) {
        // ...
    }

    // ruleid: sol-style-return-arg-fmt
    function test() returns (Contract.Struct memory b) {
        // ...
    }

    // ruleid: sol-style-return-arg-fmt
    function test() returns (Contract.Struct memory b, bool xyz) {
        // ...
    }
}

contract SemgrepTest__sol_style_doc_comment {
    function test() {
        // ok: sol-style-doc-comment
        /// Good comment

        // ok: sol-style-doc-comment
        /// Multiline
        /// Good
        /// comment
        /// @notice with natspec

        // ruleid: sol-style-doc-comment
        /**
         * Example bad comment
         */

        // ruleid: sol-style-doc-comment
        /**
         * Example
         * bad
         * Multiline
         * comment
         */

        // ruleid: sol-style-doc-comment
        /**
         * Example
         * bad
         * Multiline
         * comment
         * @notice with natspec
         */
    }
}

contract SemgrepTest__sol_style_malformed_require {
    function test() {
        // ok: sol-style-malformed-require
        require(cond, "MyContract: test message good");

        // ok: sol-style-malformed-require
        require(cond, "MyContract: test message good");

        // ok: sol-style-malformed-require
        require(!LibString.eq(_standardVersionsToml, ""), "DeployImplementationsInput: not set");

        // ok: sol-style-malformed-require
        require(cond, "MyContract: Test message");

        // ok: sol-style-malformed-require
        require(cond, "L1SB-10");

        // ok: sol-style-malformed-require
        require(cond, "CHECK-L2OO-140");

        // ok: sol-style-malformed-require
        require(cond);

        // ok: sol-style-malformed-require
        require(bytes(env_).length > 0, "Config: must set DEPLOY_CONFIG_PATH to filesystem path of deploy config");

        // ok: sol-style-malformed-require
        require(false, string.concat("DeployConfig: cannot find deploy config file at ", _path));

        // ok: sol-style-malformed-require
        require(
            _addrs[i] != _addrs[j],
            string.concat(
                "DeployUtils: check failed, duplicates at ", LibString.toString(i), ",", LibString.toString(j)
            )
        );

        // ruleid: sol-style-malformed-require
        require(cond, "MyContract: ");

        // ruleid: sol-style-malformed-require
        require(cond, "test");
    }
}

contract SemgrepTest__sol_style_malformed_revert {
    function test() {
        // ok: sol-style-malformed-revert
        revert("MyContract: test message good");

        // ok: sol-style-malformed-revert
        revert("MyContract: test message good");

        // ok: sol-style-malformed-revert
        revert("DeployImplementationsInput: not set");

        // ok: sol-style-malformed-revert
        revert("MyContract: Test message");

        // ok: sol-style-malformed-revert
        revert("L1SB-10");

        // ok: sol-style-malformed-revert
        revert("CHECK-L2OO-140");

        // ok: sol-style-malformed-revert
        revert();

        // ok: sol-style-malformed-revert
        revert("Config: must set DEPLOY_CONFIG_PATH to filesystem path of deploy config");

        // ok: sol-style-malformed-revert
        revert(string.concat("DeployConfig: cannot find deploy config file at ", _path));

        // ok: sol-style-malformed-revert
        revert(
            string.concat(
                "DeployUtils: check failed, duplicates at ", LibString.toString(i), ",", LibString.toString(j)
            )
        );

        // ruleid: sol-style-malformed-revert
        revert("MyContract: ");

        // ruleid: sol-style-malformed-revert
        revert("test");
    }
}
