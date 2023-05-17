# flags/flags.go

The cli flag schema for the indexer service

- **Example**

The following example configures a `BoolFlag` 

```go
var (
	BedrockFlag = cli.BoolFlag{
		Name:   "bedrock",
		Usage:  "Whether or not this indexer should operate in Bedrock mode",
		Required: false
		EnvVar: prefixEnvVar("BEDROCK"),
	}
)
```

- **Usage**

This `BoolFlag` can now be passed into the indexer as an `EnvVar`

```bash
BEDROCK=true go run cmd/indexer/main.go
```

or it can be specified as command line flag 

```bash
go run cmd/indexer/main.go --bedrock true
```

It will also show up in the help as follows when `go run cmd/indexer/main.go --help` is ran

```
--bedrock Whether or not this indexer should operate in Bedrock mode [$INDEXER_BEDROCK]
```

- **See also:** 

- Implementation in [flags.go](./flags.go)
- Unit tests in [flags_test.go](../cmd/indexer/README.md)
- External library docs: [urfave/cli docs](https://cli.urfave.org/v1/examples/flags)
- Cli entrypoint docs for [cmd/indexer/main.go](../cmd/indexer/README.md)

