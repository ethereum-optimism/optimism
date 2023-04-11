# Linting

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Markdown](#markdown)
- [Go](#go)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

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
yarn                   # Install dependencies
yarn lint:specs:check  # Run linter
yarn lint:specs:fix    # Fix lint issues
yarn lint:specs:toc    # Update TOC docs

# Check links
docker run --init -it -v `pwd`:/input lycheeverse/lychee --verbose --no-progress --exclude-loopback --exclude twitter.com --exclude-mail /input/README.md "/input/specs/**/*.md"
```

To check links, you'll need to install [lychee]. The [version ran in CI][lychee-ci] is 0.8.1, but
you should install lychee 0.8.2 locally with `cargo install --version 0.8.2 lychee` (there are some
reported build problems with 0.8.1).

You can install cargo (the Rust package manager) via [rustup].

[lychee]: https://github.com/lycheeverse/lychee
[lychee-ci]: https://github.com/lycheeverse/lychee-action/blob/f76b8412c668f78311212d16d33c4784a7d8762c/Dockerfile
[rustup]: https://www.rust-lang.org/tools/install

To update the TOC, we run [doctoc], installed through the dev-dependencies in `package.json`.

[doctoc]: https://github.com/thlorenz/doctoc

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
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.48.0
# run linter, add --fix option to fix problems (where supported)
golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 2m -e "errors.As" -e "errors.Is" ./...
```
