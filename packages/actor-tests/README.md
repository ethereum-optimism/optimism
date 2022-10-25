# Actor Tests

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=actor-tests-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

This README describes how to use the actor testing library to write new tests. If you're just looking for how to run test cases, check out the README [in the root of the repo](../README.md).

## Introduction

An "actor test" is a test case that simulates how a user might interact with Optimism. For example, an actor test might automate a user minting NFTs or swapping on Uniswap. Multiple actor tests are composed together in order to simulate real-world usage and help us optimize network performance under realistic load.

Actor tests are designed to catch race conditions, resource leaks, and performance regressions. They aren't a replacement for standard unit/integration tests, and aren't executed on every pull request since they take time to run.

This directory contains the actor testing framework as well as the tests themselves. The framework lives in `lib` and the tests live in this directory with `.test.ts` prefixes. Read on to find out more about how to use the framework to write actor tests of your own.

## CLI

Use the following command to run actor tests from the CLI:

```
ts-node actor-tests/lib/runner.ts -f <path-to-test-file> -c <concurrency> -r <num-runs> -t <time-to-run> --think-time <think time?>
```

You can also run `ts-node actor-tests/lib/runner.ts --help` for a full list of options with documentation.

**Arguments:**

- `path-to-test-file`: A path to the TS file containing the actor test.
- `concurrency`: How many workers to spawn.
- `run-for`: How long, in milliseconds, the worker should run.
- `think-time`: How long the runner should pause between test runs. Defaults to zero, meaning runs execute as fast as possible.

## Usage

### Test DSL

Actor tests are defined using a Mocha-like DSL. Follow along using the example below:

```typescript
import { actor, setupRun, setupActor, run } from './lib/convenience'

interface Context {
  wallet: Wallet
}

actor('Value sender', () => {
  let env: OptimismEnv

  setupActor(async () => {
    env = await OptimismEnv.new()
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom()
    const tx = await env.l2Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('0.01'),
    })
    await tx.wait()
    return {
      wallet: wallet.connect(env.l2Wallet.provider),
    }
  })

  run(async (b, ctx: Context) => {
    const randWallet = Wallet.createRandom().connect(env.l2Wallet.provider)
    await b.bench('send funds', async () => {
      const tx = await ctx.wallet.sendTransaction({
        to: randWallet.address,
        value: 0x42,
      })
      await tx.wait()
    })
    expect(await randWallet.getBalance()).to.deep.equal(BigNumber.from(0x42))
  })
})
```

#### `actor(name: string, cb: () => void)`

Defines a new actor.

**Arguments:**

- `name`: Sets the actor's name. Used in logs and in outputted metrics.
- `cb`: The body of the actor. Cannot be async. All the other DSL methods (i.e. `setup*`, `run`) must be called within this callback.

#### `setupActor(cb: () => Promise<void>)`

Defines a setup method that gets called after the actor is instantiated but before any workers are spawned. Useful to set variables that need to be shared across all worker instances.

**Note:** Any variables set using `setupActor` must be thread-safe. Don't use `setupActor` to define a shared `Provider` instance, for example, since this will introduce nonce errors. Use `setupRun` to define a test context instead.

#### `setupRun(cb: () => Promise<T>)`

Defines a setup method that gets called inside each worker after it is instantiated but before any runs have executed. The value returned by the `setupRun` method becomes the worker's test context, which will be described in more detail below.

**Note:** While `setupRun` is called once in each worker, invocations of the `setupRun` callback are executed serially. This makes `setupRun` useful for nonce-dependent setup tasks like funding worker wallets.

#### `run<T>(cb: (b: Benchmarker, ctx: T) => Promise<void>)`

Defines what the actor actually does. The test runner will execute the `run` method multiple times depending on its configuration.

**Benchmarker**

Sections of the `run` method can be benchmarked using the `Benchmarker` argument to the `run` callback. Use the `Benchmarker` like this:

```typescript
b.bench('bench name', async () => {
 // benchmarked code here
})
```

A summary of the benchmark's runtime and a count of how many times it succeeded/failed across each worker will be recorded in the run's metrics.

**Context**

The value returned by `setupRun` will be passed into the `ctx` argument to the `run` callback. Use the test context for values that need to be local to a particular worker. In the example, we use it to pass around the worker's wallet.

### Error Handling

Errors in setup methods cause the test process to crash. Errors in the `run` method are recorded in the test's metrics, and cause the run to be retried. The runtime of failed runs are not recorded.

It's useful to use `expect`/`assert` to make sure that actors are executing properly.

### Test Runner

The test runner is responsible for executing actor tests and managing their lifecycle. It can run in one of two modes:

1. Fixed run mode, which will execute the `run` method a fixed number of times.
2. Timed mode, which will will execute the `run` method as many times as possible until a period of time has elapsed.

Test lifecycle is as follows:

1. The runner collects all the actors it needs to run.

	> Actors automatically register themselves with the default instance of the runner upon being `require()`d.
2. The runner executes each actor's `setupActor` method.
3. The runner spawns `n` workers.
4. The runner executes the `setupRun` method in each worker. The runner will wait for all `setupRun` methods to complete before continuing.
5. The runner executes the `run` method according to the mode described above.

## Metrics

The test runner prints metrics about each run to `stdout` on exit. This output can then be piped into Prometheus for visualization in Grafana or similar tools. Example metrics output might looks like:

```
# HELP actor_successful_bench_runs_total Count of total successful bench runs.
# TYPE actor_successful_bench_runs_total counter
actor_successful_bench_runs_total{actor_name="value_sender",bench_name="send_funds",worker_id="0"} 20
actor_successful_bench_runs_total{actor_name="value_sender",bench_name="send_funds",worker_id="1"} 20

# HELP actor_failed_bench_runs_total Count of total failed bench runs.
# TYPE actor_failed_bench_runs_total counter

# HELP actor_step_durations_ms_summary Summary of successful bench durations.
# TYPE actor_step_durations_ms_summary summary
actor_step_durations_ms_summary{quantile="0.5",actor_name="value_sender",bench_name="send_funds"} 1278.0819790065289
actor_step_durations_ms_summary{quantile="0.9",actor_name="value_sender",bench_name="send_funds"} 1318.4640210270882
actor_step_durations_ms_summary{quantile="0.95",actor_name="value_sender",bench_name="send_funds"} 1329.5195834636688
actor_step_durations_ms_summary{quantile="0.99",actor_name="value_sender",bench_name="send_funds"} 1338.0024159550667
actor_step_durations_ms_summary_sum{actor_name="value_sender",bench_name="send_funds"} 51318.10741400719
actor_step_durations_ms_summary_count{actor_name="value_sender",bench_name="send_funds"} 40

# HELP actor_successful_actor_runs_total Count of total successful actor runs.
# TYPE actor_successful_actor_runs_total counter
actor_successful_actor_runs_total{actor_name="value_sender"} 40

# HELP actor_failed_actor_runs_total Count of total failed actor runs.
# TYPE actor_failed_actor_runs_total counter
```
