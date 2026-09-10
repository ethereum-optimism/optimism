# Utils-App

A general framework for creating typescript applications.

- Handles application lifecycle
- Metrics server configuration
- CLI configuration

## Configuration

A set of default options are provided to every instance of `App`.

```bash
--log-level <level>                     Set log level (choices: "debug", "info", "warn", "error", "silent", default: "debug", env: LOG_LEVEL)
--admin-enabled                         Enable admin API server (default: false, env: ADMIN_ENABLED)
--admin-port <port>                     Port for admin API (default: "9000", env: ADMIN_PORT)
--metrics-enabled                       Enable metrics collection (default: false, env: METRICS_ENABLED)
--metrics-port <port>                   Port for metrics server (default: "7300", env: METRICS_PORT)
```

- The [Hono](https://hono.dev/docs/getting-started/nodejs) admin api, logger, and all [commander options](https://www.npmjs.com/package/commander) are available as properties on `App`.

- An overridable method on `App` is available, `protected additionalOptions(): []Options`, used to specify any additional cli options needed.

- If enabled, the metrics server emits metrics under the `/metrics` path.

## Lifecycle

The application lifecycle follows

1. preMain(): _validate options and admin api registration prior to running app_
2. main(): _implementation of app logic_
3. shutdown() : _triggered on interruption or executed after the return of main_
   - _This MUST cause the resolution of the main promise in order for graceful shutdown. After the 5th interruption, the process is forcefully exited._
