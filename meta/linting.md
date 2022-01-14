# Linting

## Markdown

See

- [markdownlint rule reference](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md)
- [exemple .markdownlint.json file](https://github.com/DavidAnson/markdownlint/blob/main/schema/.markdownlint.jsonc)

Justification for linting rules in [.markdownlint.json](/.markdownlint.json):

- *line_length* (`!strict && stern`): don't trip up on url lines
- *no-blanks-blockquote*: enable multiple consecutive blockquotes separated by white lines
- *single-title*: enable reusing `<h1>` for content
- *no-emphasis-as-heading*: enable emphasized paragraphs

```shell
yarn           # Install dependencies
yarn lint      # Run linter
```

## Go

See

- [golangci-lint docs](https://golangci-lint.run/usage/install/#local-installation)
- [golangci-lint github](https://github.com/golangci/golangci-lint)
- [github action github](https://github.com/golangci/golangci-lint-action)

Justification for linting rules:

- *asciicheck*: no symbol names with invisible unicode and such
- *goimports*: group local and external import

```shell
# Install linter globally (should not affect go.mod)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
# run linter, add --fix option to fix problems (where supported)
golangci-lint run -E asciicheck,goimports
```
