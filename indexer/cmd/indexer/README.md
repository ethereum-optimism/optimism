# cmd/indexer/main.go

The cli entry point for the indexer.

- **Overview**

`main()` function in [cmd/indexer/main.go](./main.go) is entrypoint to program.   It parses the cli flags to pass into the [indexer](../../indexer.go)

- **Details**

We use [urfave/cli](https://cli.urfave.org/v1/getting-started/) v1 to create a new golang cli.   Note we do not use v2.

The main function does the following:

1. create a new app
2. Pass it metadata such as `Version`, `Name`, `Usage`, and `Description`
3. pass in [CLI flags](../../flags/README.md) are imported from [flags.go](../../flags/flags.go) which specifies how to configure the indexer
4. Pass in our [indexer.Main](../../indexer.go) function which runs the indexer
5. Finally call `app.Run(os.args)` on our cli app to start the app

- **See also:**

- Implementation in [main.go](./main.go)
- [urfave/cli docs](https://cli.urfave.org/v1/getting-started/)
- Indexers internal [cli flags docs](../../flags/README.md)

