# Markdown Style Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Linting](#linting)
- [Links](#links)
  - [Glossary](#glossary)
- [Internal (In-File) Links](#internal-in-file-links)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Linting

Respect the [linting rules] (you can run the linter with `pnpm lint`).

Notably:

- lines should be < 120 characters long
  - in practice, some of our files are justified at 100 characters, some at 120

[linting rules]: linting.md#markdown

## Links

In general:

- Use link references preferentially.
  - e.g. `[my text][link-ref]` and then on its own line `[link-ref]: https://mylink.com`
  - e.g. `[my text]` and then on its own line: `[my text]: https://mylink.com`
  - exceptions: where it fits neatly on a single line, in particular in lists of links
- Excepted for internal and glossary links (see below), add the link reference definition directly
  after the paragraph where the link first appears.

### Glossary

- Use links to the [glossary] liberally.
- Include the references to all the glossary links at the top of the file, under the top-level
  title.
- A glossary link reference should start with the `g-` prefix. This enables to see what links to the
  glossary at a glance when editing the specification.
  - e.g. `[g-block]: glossary.md#block`
- Example: [Rollup Node Specification source][rollup-node]

[glossary]: ../glossary.md
[rollup-node]: https://raw.githubusercontent.com/ethereum-optimism/optimistic-specs/main/specs/rollup-node.md

## Internal (In-File) Links

If linking to another heading to the same file, add the link reference directly under that heading.
This makes it easier to keep the heading and the link in-sync, and signals that the heading is being
linked to from elsewhere.
