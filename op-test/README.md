# op-test

Op-test is a test-framework that aims to achieve the following goals:
- Resource sharing: reduce duplicate setup! reduce load! Improve execution speed! Critical to CI time.
- Parametrization: hard fork choice etc. Critical to growth of protocol.
- Coordination: be effective at sharing resources between tests.
- Local-first feedback: test should be fast and quick to run in isolation.
- Native Go tests: Go test functionality is centric, not copied or avoided.
- Composition: tests should be less monolithic, and support new setups. Critical to L2 interop testing.

## Client

The client-side of op-test is integrated into the regular Go testing framework.

This is meant to make the test-execution more similar to what a regular test flow looks like,
and it reduces the feedback-loop time for test changes.

Features:
- `test.Main` to register as package through a `Main(t *testing.M)`
  - This manages client-resources across tests.
  - This writes the accumulated test-plan once, rather than per test.
  - This is what runs CLI parameter parsing, and runs tests as sub-function of a main urfave CLI action,
    so things like `--help` just work.
  - This creates a package-wide `ctx` that gets terminated upon control-C, to more gracefully terminate.
  - We can run a test package is multiple "modes":
    - `plan`: to create a test-plan
    - `server`: when running with a server.
    - other modes may be added later.
- `test.Plan(t, func(t test.Planner))` to start a test.
  - This wraps the `*testing.T` functionality as a `test.Planner`:
    - This provides common useful things like `ctx` scoped to (sub-)test and sub-test test-logger.
    - This provides parameter like `t.Select(name, params)`
    - This provides sub-test planning with `t.Plan(name, fn)`
  - This shims the `Run` to crate a `test.Executor`
- `plan.Run(name, func(t test.Executor))` to run a test
  - This sub-function doesn't execute when just in `plan` mode
  - This

### Client-side components

Components are just typed handles that wrap around a `Resource`:
  a unique identifier, created with a JSON settings object.

Instantiated resources should have no critical client-side state. State must live closer to the actual resource,
so we can share resources without conflicting usage of the resource.

Components are meant to provide a good DSL for testing, hiding details such as RPC connections,
and preferring things that describe actions.
When an "action" changes, we only have to update it in one place, rather than repeatedly in many tests.

#### Remote

Actions are communicated to a server endpoint; we use an RPC connection-pool
so we can efficiently have many parallel actions running, and don't have to re-dial new RPC clients each test-case,
or manually maintain any of the RPC clients. The `test.Main` takes care of cleanly shutting down any remaining clients.
Go-finalizers are used to get rid of any unused clients.

See `op-test/test/remote.go`.

Resources are requested through the `optest_request` RPC (also specifying the test-case name as argument),
and actions are then performed through the `optest_do` RPC.

#### Composition

See `op-test/components/$component/{types,options}.go`.

In `options.go` we define a settings struct, used to communicate what client we want with the server.
In tests we use the Go `Option` pattern however, to make component configuration more extensible.

In `types.go` we define the component interface: all DSL functions to interact with the component.

The idea is that the test `Request(opts...)`s resources, such that we can fill in the blanks as server,
and give back an implementation of the components' interface that is backed by the server through RPC.

With resources being just unique-identifiers to remote system parts,
we can pass a resource as an option to the request of another resource, such that we can enforce a direct relationship.
E.g. a L1 consensus-node may require a L1 execution-engine.

## Server

As server we run the Go tests natively, to provide access to all the Go test functionality.
*We don't impose the server as entrypoint*: running through tests selectively, and/or with debugger, is important.

The main purpose of the server is to be smarter with test-resources.
Scheduling systems and test-runs such that we maximize shared resources.

Test scheduling is a really hard problem: packing non-conflicting tests together
and executing them against a matching already-running system is challenging.

### Test Plans

A test plan simplifies the problem: it specifies what tests there are upfront,
so the server can more easily organize them.

OPEN QUESTION: should we include resource compositions (the settings-struct JSON that is constructed from options)
in the test-plan, or just selected parameters? Current parameters-only.

OPEN QUESTION: should the server ingest a structured, or a flattened, test-plan?

### Server config

Another idea to avoid complicated runtime things is to pre-configure systems in a config.
The server can load the system configs from file, and instantiate the system whenever a test matches the config.

### Running tests

To run tests in a fully scheduled manner, the server should schedule/group tests such that the tests can run
with `go test` as sub-processes (TODO: sub-process handling not implemented yet).

The `go test` command accepts arguments to filter test-cases. The server should select only the cases intended to run:
i.e. parameters that are serviceable and sub-tests that are meant to be scheduled.

The `go test` command is essentially extended by the `test.Main` CLI handling:
after a `--` separator the native go test flags stop, and the op-test CLI arguments start.

The server should specify the server-RPC endpoint through the `--server` CLI argument,
so the client-side can communicate to the server that is running the tests.

As dev, you may run the server stand-alone, and run the Go tests standalone,
manually specifying the endpoint through the CLI to hook it up to the server.
This then allows you to customize the `go test` command, and debug the test in a local IDE.

### Server-side components

Components are meant to run within the server.
Upon receiving an `optest_request` RPC call, the server should match the request with
a component in the system that was allocated to be used by the test.

The client-side resource should then be registered, such that it can be closed when the test-case exits.

We then serve `optest_do` to handle changes to the resource.

#### Resource kinds

Some components may be backed by different kinds, independent of the test-case.

The three main kinds are: "live", "managed", "instant".

In some cases it doesn't make muc of a difference: chains are "resources" to,
but just hold runtime configuration for other components to use.

##### Live

The "live" kind may be a part of a devnet, an actual testnet, or maybe even a mainnet.
This kind may be requested, but not customized during testing.

Ideally we can specify future upgrade-validation scripts as tests,
and simply point them at a system with live resources.
This way we get more coverage of the upgrade-validation, in CI tests of different kinds,
and apply it across different configured systems easily.

##### Managed

This kind is most similar to the clients in the `System` in the op-e2e test suite:
an in-process version of an op-stack component, that can be configured, started and shut-down as necessary.

Shadow-forks may be created by using some "live" resources,
and attaching a "managed" resource against it, to temporarily test a divergence of what the live system may look like.

##### Instant

This kind is most similar to the clients in the op-e2e action-tests.
Slimmed-down synchronous versions of op-stack components, that don't operate on their own,
but rather step through system-changes instantly, when requested to do so.

By combining this with managed or live resources we can reduce the work-load of tests a lot,
and step through system changes with more fine-grained control:
e.g. an instant L1 consensus-layer that builds the blocks on demand,
or an instant L2 rollup node to test a managed L2 execution engine faster.

## Test-flow

1. Server spins up, is aware of possible resource configurations, but lazily creates systems when it needs them.
2. The server is requested to load and execute one or more test-plans.
3. Test-plans are generated by running test packages with `--mode=plan` op-test flag.
4. A bundle of test plans (one plan per package) is sent to the server
5. We iterate through the bundle, and for each applicable system, we register that the sub-test could use it
   - Note that a sub-test runs a specific set of parameters. If it runs, we filled that variant of the sub-test.
     The same sub-test code-path, with different parameters, may be defined separately in the plan.
6. Workers run/maintain systems. Available workers will pick the most in-demand system to run.
7. A worker iterates through the tests that registered to use its current system.
   - When a test is accepted, it is unregistered from other systems (remember, specific set of parameters).
     This relieves pressure from systems before we spin them up.
8. A worker prioritizes to take tests from the same package, to only run the binary once
9. When running a test-binary, we filter test-names to avoid Skip() calls, and apply a global set of parameters.

## TODO

The above set of functionality is not yet complete. Development of op-test has been on-and-off throughout interop,
with some design set-backs (many requirements, interactions between those requirements) and many distractions.

What has been done:
- Found a way to properly integrate urfave-CLI with go tests, through `test.Main`
- Instrumented `*testing.T` to add test-context and logger re-use.
- Instrumented `*testing.T` to add parameter selection.
- Instrumented `*testing.T` to separate test-execution.
- Implemented parameter-selector, that can load preferences from CLI, and then apply this upon `Select` calls.
- Implemented test-plan format and JSON output, including test-package import-path based naming.
- Service setup with CLi flags, CLI-config loading, and lifecycle for op-test server.

What has been tried, but failed:
- Test-parameters through context-values -> not easy to output plans for
- Unsafe-inspect/modify of `*testing.M` to reschedule tests -> package scope still limits scheduling,
  and Go-test parallel workers are in the way of smarter test execution still.
- Test-selection in-place, no sub-tests -> `runtime.Goexit` (same as what a `t.Skip` or `t.Fail` uses)
  upon `t.Select` to stop the routine, and then re-enter with all options, works as possible solution.
  It adds complexity however, and we likely shouldn't hide nesting of tests.
- Dynamic parameter-group matching -> sorting groups of parameters (by sorted keys and then compared values),
  and then looking up the "closest compatible" set of parameters, adds a lot of complexity to testing.
  After starting this I realized that resource-configuration is more important than extraordinary parametrization.
  If we add parameters, just linearly comparing against all running systems is probably fast enough.
- Directly hooking up `Request` of component to the local instantiation.
  To meet all requirements (resource sharing primarily) we need to run the component on server-side.
  The current `Request(t, opts...)` stubs should be rewired to provide RPC bindings
  around `Resource` identifiers instead.

What has been tried, but likely to fail / need of redo:
- Experimented with demo-test to try and get components running. Still more work to do on getting components to run remotely.
- Flesh out the components: while started with porting over of the op-e2e components into "managed" components,
  untangling it from the monolithic `System` struct is not easy.
  The action-test components may be better contained and thus easier to port over to op-test "instant" type.
  The "live" version likely should not be prioritized, as it is the least flexible and hardest to iterate on.
- Start of Server worker / system / plan organization. Experimentation with this has show that managing these
  different running things is challenging. See below todo-section for a proposal of how it could work.
- Client-side execution needs to reproduce the same parameters.
  We should only have to run a test-package binary with one set of parameters at a time,
  and limited number of test-cases. Loading the test-plan back into the client is possible currently,
  but not the ideal solution. Applying the chosen set of parameters will naturally limit the tests to what is needed.
  A go test-filter should be applied to limit the execution to test-cases that matcht the right parameters / resources.

What needs to be done:
- Server RPC backend to serve:
  - `optest_executePlans`: take test-plan(s) and schedule/execute them.
    - Load test-plan JSON file(s) by file matching pattern.
    - Find applicable systems (by matching parameters and possible resource settings),
      tag each system with the test-cases that can run with said system. There may be duplicates.
    - Make worker-routines (up to N workers) select the most in-demand system, and run test-cases for that system.
      Upon start of a test-case, clear it from running on any other workers / systems.
    - Repeat until the plan is fully executed.
  - `optest_request`: to create resource by component type, from supplied settings JSON struct.
    - Resources should be added to resource-manager,
      so that cross-referenced resources can find each others instantiations.
  - `optest_do`: to apply actions to resources.
- Components need to be fleshed out:
  - `Request` should map `Options` list to a settings struct,
    and then return a `{uuid, settings}` that fills the interface but panics if any method is called in the plan-phase.
  - Each component should have a `FromSettings(system, settings)` for the server to instantiate it,
    against a system, or return `nil` if the system is not able to support the settings.
  - The "managed", "instant", "live" version of each component.
- Server resource manager should shut down systems / resources when there are no tests left
  (i.e. when test-case-list for potential usage of that system/resource is empty).
- Extend the functionality of test parameters; L1 fork and L2 fork parameters are already there,
  but integration of those, and new parameters, into the code of components is important.
- Merge test-results of the server into one output, specifically for CI usage.

