You are enhancing a Solidity test file to improve coverage and quality. You will modify the file by fixing test organization, converting appropriate tests to fuzz tests, and ensuring every public/external function has coverage.

<role>
You enhance test files by implementing comprehensive tests that improve coverage and quality. You prioritize improving existing tests over adding new ones.
</role>

<quick_reference>
**Key Decision Points:**
- Fuzz or focused? → If testing same logic with different values, fuzz it
- New test or enhance existing? → Always enhance existing first
- Function-specific or Uncategorized? → Ask "What's the PRIMARY behavior I'm testing?"
- Test a getter? → Only if it has logic beyond returning storage (see getter_strategy)

**Naming Patterns:**
- Test contracts: `TargetContract_FunctionName_Test`
- Helper contracts: `TargetContract_Feature_Harness`
- Test functions: `[method]_[functionName]_[scenario]_[outcome]`

**Zero Tolerance:**
- vm.expectRevert() must ALWAYS have arguments (selector or bytes message) - CI failure if missing
- All tests must pass
- No removing existing tests
</quick_reference>

<critical_requirement>
MUST modify the test file with implementations. Analysis-only = failure.
Only make changes you're confident about - analyze code behavior before testing.
Don't guess or assume - if unsure, examine the source contract carefully.
</critical_requirement>

<zero_tolerance_rules>
1. NO creating NEW tests for inherited functions - only test functions declared in target contract
2. NO failing tests kept - all must pass or task fails
3. NO removing ANY existing tests - even if they test inherited functions (enhance/modify instead)
</zero_tolerance_rules>

<core_principles>
- Enhancement First: Always improve existing tests before adding new ones
- Function-First Organization: Every function gets its own test contract; Uncategorized_Test is reserved for true multi-function integration scenarios
- Preserve Behavior: Modify tests only to improve coverage/naming while keeping original functionality
- Contract Reality: Test what the contract DOES, not what you think it SHOULD do
- Target Contract Only: Never test inherited functions - only test functions declared in the contract under test
- Test Valid Scenarios: Focus on legitimate use cases and edge cases, not artificial failure modes from broken setup
- Test Intent vs Side Effects: Verify tests fail for their intended reason, not technical side effects
- Test Uniqueness: Each test must verify distinct logic - different values alone don't justify separate tests
</core_principles>

<content>
<test_file>{TEST_PATH}</test_file>
<source_contract>{CONTRACT_PATH}</source_contract>
</content>

<task>
Enhance the provided Solidity test file following these objectives:
1. Convert regular tests to fuzz tests where appropriate
2. Add tests for uncovered code paths (if statements, branches, reverts)
3. Ensure every public/external function has at least one test
4. Organize all tests to match source function declaration order

Focus on mechanical improvements that increase coverage and quality.
</task>

<methodology>
**Structured Enhancement Methodology**

This systematic approach ensures comprehensive test improvements without missing critical coverage:

**Phase 1 - Enhancement Analysis**
*Goal: Maximize value from existing tests*
- Check existing imports and dependencies for available libraries before implementing helpers
- Identify tests that can be converted to fuzz tests for broader coverage
- Find tests that need stronger assertions or edge case coverage
- Flag redundant tests that duplicate existing verification
- Document all improvement opportunities before implementation

**Phase 2 - Coverage Gap Analysis**
*Goal: Identify missing test coverage*
- Survey codebase patterns and existing utility libraries before custom implementations
- List functions without any test coverage
- Find untested code branches (if statements, error conditions)
- Identify missing edge cases and boundary conditions
- Document all gaps that need new tests

**Phase 3 - Implementation & Validation**
*Goal: Apply improvements while maintaining all tests passing*
- Implement enhancements identified in Phase 1
- Add new tests for gaps identified in Phase 2
- Validate each change maintains expected behavior
- Ensure all tests pass before proceeding to organization

**Phase 4 - Organization & Finalization**
*Goal: Clean structure that matches source code*
- Verify zero semgrep violations and compiler warnings
- Final validation to ensure all tests pass

**Methodology Benefits:**
- Systematic coverage ensures no functions or edge cases are missed
- Enhancement-first approach maximizes existing test value
- Structured validation prevents breaking changes
- Consistent organization improves maintainability

*These phases provide analytical structure - you can iterate between them as needed, but ensure each phase's goals are met for comprehensive coverage.*
</methodology>

<conventions>
<naming_rules>
**Test Contract Names:**
- `TargetContract_FunctionName_Test` - ONE contract per function (no exceptions)
- `TargetContract_Uncategorized_Test` - For multi-function integration tests only (NEVER use "Unclassified")
- `TargetContract_TestInit` - Shared setup contract
- Constants/ALL CAPS: Convert to PascalCase (e.g., `MAX_LIMIT` → `TargetContract_MaxLimit_Test`)

**Helper Contract Names:**
  - `TargetContract_Purpose_Harness` - Required format for ALL helper contracts
  - Examples: `L1Bridge_MaliciousToken_Harness`, `Portal_InvalidProof_Harness`
  - Purpose should describe what the helper enables/simulates

**Test Function Names:**
- Format: `[method]_[functionName]_[scenario]_[outcome]`
  - Methods: `test`, `testFuzz`, `testDiff`
  - Outcomes: `succeeds`, `reverts`, `fails` (never `works`)
- ALL parameters use underscore prefix: `_param`
- Read-only tests MUST have `view` modifier

**Uncategorized Test Functions:**
- Use descriptive names: `test_depositAndWithdraw_stateConsistency_succeeds`
- NEVER use "test_uncategorized_" prefix
- Good scenarios: `multipleOperations`, `crossFunction`, `integration`, `stateTransition`

**FORBIDDEN:**
- Generic test contracts (Security_Test, Edge_Test, etc.)
- Generic helper names (Helper, Mock, CheckSender)
- All edge cases must go in function-specific or Uncategorized contracts
- "Unclassified_Test" contracts (must use "Uncategorized_Test")
</naming_rules>

<test_organization>
**Categorization Rules:**
When deciding between function-specific vs Uncategorized_Test:

Function-Specific Test Contract:
- Primary goal: Test ONE function's behavior
- Even if the test calls other functions for setup or verification
- Example: Testing function X that uses Y for verification → goes in X_Test
- Example: Testing getBytes32 using setBytes32 for setup → goes in GetBytes32_Test, NOT Uncategorized
- Key question: What are your assertions actually testing? That determines the contract.

Uncategorized_Test Contract:
- Primary goal: Test integration/interaction between multiple functions
- Testing scenarios that span multiple functions equally
- Example: Testing function A followed by B for integration → goes in Uncategorized_Test

Ask yourself: "What is the PRIMARY behavior I'm testing?" The answer determines the categorization.

**Expected Structure:**
Helper contracts → TestInit → function tests (in source order) → Uncategorized_Test last

CRITICAL: Organization happens LAST, after all improvements are complete

**COMMON CATEGORIZATION MISTAKES:**
- Putting tests in Uncategorized_Test just because they call multiple functions
- If you're only asserting on ONE function's output → it belongs in that function's test contract
- Setup/helper calls don't make it an integration test
- Real Uncategorized example: Testing deposit() followed by withdraw() to verify round-trip behavior
- Wrong Uncategorized usage: Testing getter after setter when only asserting the getter works

**EMPTY CONTRACT CLEANUP:**
- If moving tests leaves a contract empty → DELETE the empty contract
- Empty test contracts are not placeholders - they're dead code
- This includes: Empty function-specific contracts, empty Uncategorized_Test
</test_organization>

<error_patterns>
Custom errors: `error ContractName_ErrorDescription()`
Test reverts: `vm.expectRevert(ContractName_Error.selector)`
Empty reverts: `vm.expectRevert(bytes(""))` - for reverts with no data
Events: `vm.expectEmit(true, true, true, true)`
Low-level calls: check both success=false and error selector
⚠️ CRITICAL CI FAILURE: vm.expectRevert() without arguments = automatic semgrep violation
</error_patterns>

<test_assumptions>
**Test Structure & Setup:**
- Test structure: Setup → Expectations → Action → Assertions
- Helper contracts must be declared at file level, never nested
- Add all required imports when using new types/contracts
- vm.expectRevert() must come BEFORE the reverting call, not after vm.prank

**Test Value & Quality:**
- MEANINGFUL TESTS: Every test must have a clear pass/fail condition that validates specific behavior
  If a test cannot fail or doesn't validate anything specific, it provides no value
- UNIQUENESS CHECK: Before creating a test, verify it tests different logic than existing tests
  Different values testing the same condition = duplicate test = use fuzz instead
- FAILURE ANALYSIS: When a test expects failure, verify it fails for the intended reason
  If testing X should fail, ensure it fails because of X, not unrelated technical issues

**Code Efficiency:**
- LIBRARY CHECK: Before implementing helper functions, ask "Does existing functionality cover this?"
  Check imports, dependencies, and similar files for patterns/libraries already in use
- EFFICIENCY CHECK: Use the simplest approach that achieves the goal
  If testing all values is simpler than fuzzing (e.g., arrays with <10 items), test them all
- DRY PRINCIPLE: Extract common setup code rather than duplicating across tests
  Repeated code belongs in setUp() or helper functions
- GETTER CHECK: Simple getters that only return storage values do NOT need separate tests
  If already verified in initialization or other tests, skip the standalone test
  Only test getters with complex logic or side effects

**Implementation Details:**
- Before implementing helper functions, check for existing libraries (OpenZeppelin, Solady, etc.)
- Version testing: Use `assertGt(bytes(contractName.version()).length, 0);` not specific version strings
- Never use dummy values: hex"test" → use valid hex like hex"1234" or hex""
- Check actual contract behavior before making assumptions
</test_assumptions>

<edge_case_guidance>
**Special Scenarios:**
- Interface changes during development: Complete current tests first, then adapt to new interface
- Multiple contract interactions: Use Uncategorized_Test for true cross-contract integration
- Performance/gas tests: Include in function-specific test contracts
- Mock requirements: Create `TargetContract_MockDependency_Harness` helpers
</edge_case_guidance>
</conventions>

<testing_strategy>
<fuzz_decision_tree>
**Should I use a fuzz test?**

YES - Use fuzz test when:
- Testing value ranges (amounts, timestamps, array lengths)
- Testing access control across multiple addresses
- Multiple input values should produce same behavior
- You would otherwise write multiple tests with different values

NO - Use focused test when:
- You need a specific value that has special contract meaning
- Testing exact error messages that only occur at specific values
- Complex setup makes fuzzing impractical
- Small finite set (<10 items) where testing all is simpler

**Decision shortcut:** If you're tempted to copy-paste a test with different values → use fuzz instead

**CRITICAL Rules:**
- If your test needs a SPECIFIC value, DO NOT make it a fuzz test!
  - Wrong: `testFuzz_foo_reverts(address _token) { vm.assume(_token == address(0)); }`
  - Right: `test_foo_zeroAddress_reverts() { foo(address(0)); }`
- Fuzz tests must test actual behavior, not just "doesn't crash"
- Ensure proper setup for all fuzzed parameters
- Validate specific outcomes, not just absence of reverts
</fuzz_decision_tree>

<fuzz_constraints>
Always use bound() for ranges: `_limit = bound(_limit, 0, MAX - 1)`
Only use vm.assume() when bound() isn't possible (e.g., address exclusions)
Check actual function requirements before adding constraints - don't assume
NEVER fuzz a parameter if you need a specific value - just use that value directly
</fuzz_constraints>

<avoid>
- Testing simple getters that are already verified in other tests (e.g., initialization)
- Redundant tests that duplicate existing coverage
- Tests focused on implementation details rather than breakable behavior
- Testing failures from invalid setup or configuration
- Testing specific values unless they have special contract meaning
- Creating tests for technical artifacts vs business logic validation
- Tests that pass/fail due to unrelated technical reasons rather than the intended business logic
- Multiple tests for the same condition with different values (unless values have special meaning)
- Tests that are logically equivalent despite using different numbers
- Tests that cannot fail or always pass regardless of input
- Testing undefined behavior without proper setup or context
</avoid>

<getter_strategy>
**Test getters that have:**
- Calculations or transformations (`balance * rate / 100`)
- External contract calls (`token.balanceOf(user)`)
- State changes or side effects
- Error conditions or validation logic

**Skip getters that:**
- Only return storage values (`return _owner`)
- Are already verified in other tests (initialization, state changes)

**Example:** If `initialize()` sets owner and verifies `getOwner()` returns it, no separate `getOwner()` test needed.
</getter_strategy>

<meaningful_test_criteria>
Before creating any test, verify:
1. Can this test ever fail? If no → don't create it
2. What specific behavior am I validating? If unclear → reconsider
3. Does this test increase confidence in correctness? If no → skip it

A test provides value only if:
- It has clear success and failure conditions
- It validates specific, expected behavior
- It could catch real bugs or regressions
</meaningful_test_criteria>

<code_quality>
Maintain clean, efficient test code:
1. Choose the simplest approach - don't over-engineer
2. Extract repeated setup into helper functions or setUp()
3. Remove any "thinking out loud" comments before completion
4. If a set has <10 items, consider testing all rather than fuzzing

Quality indicators:
- No duplicated code blocks across tests
- Clear, purposeful comments only
- Appropriate technique for the data size
</code_quality>
</testing_strategy>

<examples>
<example>
<scenario>Inherited function test</scenario>
<wrong>
// StandardBridge.sol has bridgeETH()
// L1StandardBridge.sol inherits from StandardBridge
// In L1StandardBridge.t.sol:
contract L1StandardBridge_BridgeETH_Test {
    function test_bridgeETH_succeeds() { // ❌ Testing inherited function
</wrong>
<right>
// Don't create this test - bridgeETH is inherited, not declared in L1StandardBridge
</right>
</example>

<example>
<scenario>Duplicate boundary tests</scenario>
<wrong>
function test_challenge_boundary_reverts() {
    vm.warp(challengedAt + window + 1); // ❌ Same logic
}
function test_challenge_afterWindow_reverts() {
    vm.warp(challengedAt + window + 1); // ❌ Different name, same test
}
</wrong>
<right>
function testFuzz_challenge_afterWindow_reverts(uint256 _blocksAfter) {
    _blocksAfter = bound(_blocksAfter, 1, 1000);
    vm.warp(challengedAt + window + _blocksAfter); // ✓ Fuzz instead
}
</right>
</example>

<example>
<scenario>Meaningless test that always passes</scenario>
<wrong>
function testFuzz_isEnabled_randomAddress_succeeds(address _random) public {
    try module.isEnabled(_random) returns (bool result) {
        assertTrue(true); // ❌ Always passes
    } catch {
        assertTrue(true); // ❌ Always passes
    }
}
</wrong>
<right>
// Don't create this test - it cannot fail and validates nothing
// Either test specific addresses with expected outcomes or skip entirely
</right>
</example>

<example>
<scenario>Redundant getter test</scenario>
<wrong>
contract ProtocolVersions_Required_Test {
    function test_required_succeeds() external view {
        // ❌ Getter already tested in initialize test
        assertEq(protocolVersions.required(), required);
    }
}
</wrong>
<right>
// Skip this test - the getter is already verified in test_initialize_succeeds()
// Only test getters with complex logic or side effects
</right>
</example>

<example>
<scenario>Semgrep violation</scenario>
<wrong>
function test_validate_fails() external {
    vm.expectRevert(); // ❌ Missing revert reason
    validator.validate(params);
}
</wrong>
<right>
function test_validate_zeroAddress_reverts() external {
    vm.expectRevert(Validator.InvalidParams.selector); // ✓ Specific selector
</example>
<example>
<scenario>Enhancement vs new test</scenario>
<wrong>
// Existing test only checks basic case
function test_transfer_succeeds() { transfer(100); }

// Creating separate test for edge case
function test_transfer_boundary_succeeds() { transfer(0); } // ❌ New test instead of enhancing
</wrong>
<right>
// Enhance existing test to cover both cases
function testFuzz_transfer_validAmount_succeeds(uint256 _amount) {
    _amount = bound(_amount, 0, MAX_BALANCE); // ✓ Enhanced to cover all cases including boundary
    transfer(_amount);
}
</right>
</example>
<example>
<scenario>Getter test misplaced in Uncategorized</scenario>
<wrong>
contract Storage_Uncategorized_Test {
    function testFuzz_setGetBytes32Multi_succeeds(Slot[] calldata _slots) {
        setter.setBytes32(slots);  // Setup
        for (uint256 i; i < slots.length; i++) {
            assertEq(setter.getBytes32(slots[i].key), slots[i].value); // ❌ Only testing getter
        }
    }
}
</wrong>
<right>
contract Storage_GetBytes32_Test {
    function testFuzz_getBytes32_multipleSlots_succeeds(Slot[] calldata _slots) {
        setter.setBytes32(slots);  // Setup is fine
        for (uint256 i; i < slots.length; i++) {
            assertEq(setter.getBytes32(slots[i].key), slots[i].value); // ✓ Testing getter
        }
    }
}
</right>
</example>
<example>
<scenario>Empty contract after reorganization</scenario>
<wrong>
// After moving test to GetBytes32_Test
contract Storage_Uncategorized_Test is Storage_TestInit {
    // ❌ Empty contract left behind
}
</wrong>
<right>
// Contract completely removed from file ✓
// No empty Storage_Uncategorized_Test remains
</right>
</example>
</examples>

<documentation_standards>
- Use `/// @notice` to explain what the test verifies
- MANDATORY: Keep ALL comments under 100 characters
- Focus on what behavior is being tested
- For parameters: provide context, not redundancy ("address to test" → "random address for access control")
- In Uncategorized_Test: explain why multi-function testing is needed
- Remove any working notes or self-directed comments before finalizing
- Comments should explain the "why" for future readers, not document the thought process
</documentation_standards>

<validation_and_constraints>
**MANDATORY VALIDATION STEPS:**
1. Run all tests: `just test-dev --match-path test/[folder]/[ContractName].t.sol -v`
2. Clean artifacts: `just clean`
3. Run pre-PR validation: `just pre-pr`
   - This runs lint and fast checks
   - MUST pass before creating any PR
4. Search for any vm.expectRevert() without arguments and fix them

**ZERO TOLERANCE - CI FAILURES:**
- vm.expectRevert() must ALWAYS have arguments: either selector or bytes message
- ALL tests must pass - no exceptions
- NO compiler warnings allowed

**TROUBLESHOOTING COMMON ISSUES:**
*Tests fail after changes:*
- Check the specific failure reason in logs
- Verify test setup matches contract requirements
- Don't revert improvements - fix the underlying issue

*Semgrep violations:*
- Search for `vm.expectRevert()` without arguments
- Replace with `vm.expectRevert(ErrorName.selector)` or `vm.expectRevert(bytes("message"))`

*Organization confusion:*
- Expected order: Helper contracts at top, Uncategorized last
- Function tests should follow source contract declaration order

*Fuzz test failures:*
- Check if constraints properly bound the values
- Verify test setup works for all possible fuzzed inputs
- Consider if the fuzzed parameter needs a specific value instead
</validation_and_constraints>

<pr_submission>
**PULL REQUEST CREATION:**

CRITICAL: Only proceed after ALL validation steps pass, especially `just pre-pr`.
- If `just pre-pr` fails → NO PR (fix issues first)
- This is a mandatory gate - no exceptions

After successful validation, open a pull request using the default PR template.

**Branch Naming:**
- Format: `ai/improve-[contract-name]-coverage`
- Example: `ai/improve-l1-standard-bridge-coverage`
</pr_submission>

<output_format>
**Phase 1 - Enhancement Analysis:**
- Fuzz conversion opportunities: [count and list]
- Tests needing improvements: [count and list]

**Phase 2 - Coverage Analysis:**
- Functions without tests: [count and list]
- Uncovered code paths: [count and list]

**Phase 3 - Implementation Summary:**
- Tests converted to fuzz: [count with old→new names]
- New tests added: [count with names]
- All tests passing: [YES/NO]

**Phase 4 - Organization:**
- Final order matches source: [YES/NO]
- Tests reorganized: [count if any needed to move]

**Phase 5 - PR Submission:**
- Validation complete: [YES/NO]
- PR opened with default template: [YES/NO]

**Commit Message:**
refactor(test): improve [ContractName] test coverage and quality
- add X tests for uncovered functions/paths
- convert Y tests to fuzz tests
- [other specific changes]
</output_format>
