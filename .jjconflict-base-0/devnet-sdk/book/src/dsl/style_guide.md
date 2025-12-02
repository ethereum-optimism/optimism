# DSL Style Guide

This style guide outlines common patterns and anti-patterns used by the testing DSL. Following this guide not only
improves consistency, it helps keep the separation of requirements (in test files) from implementation details (in DSL
implementation), which in turn ensures tests are maintainable even as the number of tests keeps increasing over time.

## Entry Points

What are the key entry points for the system? Nodes/services, users, contracts??

## Action Methods

Methods that perform actions will typically have three steps:

1. Check (and if needed, wait) for any required preconditions
2. Perform the action, allowing components to fully process the effects of it
3. Assert that the action completed. These are intended to be a sanity check to ensure tests fail fast if something
   doesn't work as expected. Options may be provided to perform more detailed or specific assertions

## Verification Methods

Verification methods in the DSL provide additional assertions about the state of the system, beyond the minimal
assertions performed by action methods.

Verification methods should include any required waiting or retrying.

Verification methods should generally only be used in tests to assert the specific behaviours the test is covering.
Avoid adding additional verification steps in a test to assert that setup actions were performed correctly - such
assertions should be built into the action methods. While sanity checking setup can be useful, adding additional
verification method calls into tests makes it harder to see what the test is actually intending to cover and increases
the number of places that need to be updated if the behaviour being verified changes in the future.

### Avoid Getter Methods

The DSL generally avoids exposing methods that return data from the system state. Instead verification methods are
exposed which combine the fetching and assertion of the data. This allows the DSL to handle any waiting or retrying
that may be necessary (or become necessary). This avoids a common source of flakiness where tests assume an asynchronous
operation will have completed instead of explicitly waiting for the expected final state.


```go
// Avoid: the balance of an account is data from the system which changes over time
block := node.GetBalance(user)

// Good: use a verification method
node.VerifyBalance(user, 10 * constants.Eth)

// Better? Select the entry point to be as declarative as possible
user.VerifyBalacne(10 * constants.Eth) // implementation could verify balance on all nodes automatically
```


Note however that this doesn't mean that DSL methods never return anything. While returning raw data is avoided,
returning objects that represent something in the system is ok. e.g.

```go
claim := game.RootClaim()

// Waits for op-challenger to counter the root claim and returns a value representing that counter claim
// which can expose further verification or action methods.
counter := claim.VerifyCountered()
counter.VerifyClaimant(honestChallenger)
counter.Attack()
```

## Method Arguments

Required inputs to methods are specified as normal parameters, so type checking enforces their presence.

Optional inputs to methods are specified by a config struct and accept a vararg of functions that can update that struct.
This is roughly inline with the typical opts pattern in Golang but with significantly reduced boilerplate code since
so many methods will define their own config. With* methods are only provided for the most common optional args and
tests will normally supply a custom function that sets all the optional values they need at once.

## Logging

Include logging to indicate what the test is doing within the DSL methods.

Methods that wait should log what they are waiting for and the current state of the system on each poll cycle.

## No Sleeps

Neither tests nor DSL code should use hard coded sleeps. CI systems tend to be under heavy and unpredictable load so
short sleep times lead to flaky tests when the system is slower than expected. Long sleeps waste time, causing test runs
to be too slow. By using a waiter pattern, a long timeout can be applied to avoid flakiness, while allowing the test to
progress quickly once the condition is met.

```go
// Avoid: arbitrary delays
node.DoSomething()
time.Sleep(2 * time.Minute)
node.VerifyResult()

// Good: build wait/retry loops into the testlib method
node.DoSomething()
node.VerifyResult() // Automatically waits
```

## Test Smells

"Smells" are patterns that indicate there may be a problem. They aren't hard rules, but indicate that something may not
be right and the developer should take a little time to consider if there are better alternatives.

### Comment and Code Block

Where possible, test code should be self-explanatory with testlib method calls that are high level enough to not need
comments explaining what they do in the test. When comments are required to explain simple setup, it's an indication
that the testlib method is either poorly named or that a higher level method should be introduced.

```go
// Smelly: Test code is far too low level and needs to be explained with a comment
// Deploy test contract
storeProgram := program.New().Sstore(0, 0xbeef).Bytes()
walletv2, err := system.NewWalletV2FromWalletAndChain(ctx, wallet, l2Chain)
require.NoError(t, err)
storeAddr, err := DeployProgram(ctx, walletv2, storeProgram)
require.NoError(t, err)
code, err := l2Client.CodeAt(ctx, storeAddr, nil)
require.NoError(t, err)
require.NotEmpty(t, code, "Store contract not deployed")
require.Equal(t, code, storeProgram, "Store contract code incorrect")

// Good: Introduce a testlib method to encapsulate the detail and keep the test high level
contract := contracts.SStoreContract.Deploy(l2Node, 0xbeef)
```

However, not all comments are bad:

```go
// Good: Explain the calculations behind specific numbers
// operatorFeeCharged = gasUsed * operatorFeeScalar == 1000 * 5 == 5000
tx.VerifyOperatorFeeCharged(5000)
```
