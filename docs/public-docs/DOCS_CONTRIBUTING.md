# Contributing to Optimism Docs

Thanks for taking the time to contribute! ❤️

## Table of Contents
- [Contributing to Optimism Docs](#contributing-to-optimism-docs)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Development Setup](#development-setup)
  - [Contributing Process](#contributing-process)
    - [File Architecture](#file-architecture)
    - [Content Guidelines](#content-guidelines)
    - [Shared snippets](#shared-snippets)
    - [Local Testing](#local-testing)
    - [Nav and redirect lint](#nav-and-redirect-lint)
  - [Pull Request Process](#pull-request-process)
    - [Before Submitting](#before-submitting)
    - [Submission Guidelines](#submission-guidelines)
    - [Review Process](#review-process)
  - [Code of Conduct](#code-of-conduct)
  - [Additional Ways to Contribute](#additional-ways-to-contribute)

## Overview

Optimism's documentation is open-source and hosted on GitHub in the [`ethereum-optimism/optimism`](https://github.com/ethereum-optimism/optimism) monorepo under [`docs/public-docs`](https://github.com/ethereum-optimism/optimism/tree/develop/docs/public-docs). The documentation is rendered at [docs.optimism.io](https://docs.optimism.io). You can contribute either by:
- Forking the `optimism` repository and working locally
- Using the "Suggest edits" button on any documentation page for smaller updates

All contributions, pull requests, and issues should be in English at this time. 

## Getting Started

### Prerequisites
- Basic knowledge of Git and GitHub
- Familiarity with Markdown
- Understanding of technical documentation principles
- Node.js and npm installed

### Development Setup
1. Install [pnpm](https://pnpm.io/installation)
2. Run `pnpm i` to install dependencies
3. Run `pnpm dev` to start development server
4. Visit [localhost:3000](http://localhost:3000)

You can now start changing content and see the website updated live each time you save a new file. 🤓

## Contributing Process

### File Architecture

See the [mintlify docs](https://www.mintlify.com/docs/organize/navigation).

**Warning**: The `public` folder contains `robots.txt` and `sitemap.xml` for SEO purposes. These files are maintained by the Documentation team only.

### Content Guidelines
We use [mintlify](https://www.mintlify.com/docs) to power our docs.

Please refer to our comprehensive [Style Guide](https://docs.optimism.io/op-stack/contribute/style-guide) ([source](op-stack/contribute/style-guide.mdx)) for detailed formatting instructions.

Before adding new content, check the [Content Guide](https://docs.optimism.io/op-stack/contribute/content-guide) ([source](op-stack/contribute/content-guide.mdx)) — it defines what belongs on docs.optimism.io, the canonical home for each content type, and how to mark third-party content.

### Shared snippets

Every fact should be stated once and referenced everywhere else. When the same
block of prose (a warning, a disclaimer, a preamble) appears on more than one
page, extract it into a [Mintlify snippet](https://www.mintlify.com/docs/create/reusable-snippets)
instead of copying it. `snippets/` is Mintlify's reserved folder: files there
never render as standalone pages.

**Placement rule**

- Machine-written snippets live under `snippets/generated/` and carry
  do-not-edit provenance headers. They are owned by generation pipelines; hand
  edits will fail drift checks.
- Hand-maintained shared prose lives at the `snippets/` root, in a kebab-case
  file, with a leading `{/* ... */}` comment naming its purpose, its usage, and
  its consumer pages (precedents: `snippets/third-party-content.mdx`,
  `snippets/op-geth-eol.mdx`). Keep the consumer list in the header comment up
  to date when you add or remove an import.

**Import forms**

- Plain include: the page imports the snippet as a default export and renders
  it as a component.

  ```mdx
  import OpGethEol from "/snippets/op-geth-eol.mdx"

  <OpGethEol />
  ```

- Parameterized include: the snippet exports an arrow-function component taking
  props, imported by name (precedent: `snippets/normative-spec.mdx`).

  ```mdx
  import { NormativeSpec } from "/snippets/normative-spec.mdx"

  <NormativeSpec what="..." title="..." href="..." />
  ```

**Extraction rule**

Only extract verbatim or near-verbatim blocks. Audience-specific framing stays
on the page, outside the snippet. If the copies have drifted, reconcile the
wording against the source of truth first, then extract; do not parameterize
prose that ought to read differently per audience.

### Local Testing

Follow these [docs](https://www.mintlify.com/docs/installation) for local changes.

### Nav and redirect lint

`docs.json` (navigation + redirects) is a guarded artifact. Two deterministic
checks apply to every change that touches `docs/public-docs/`. They are enforced
by a [Mintlify automation](https://www.mintlify.com/docs/automations) that runs
on content updates and proposes review-gated fixes, and they should be run
locally before pushing (see below):

- **Nav validator** (`scripts/lint/validate-nav.ts`): every `.mdx` on disk must
  be reachable from `docs.json` navigation or explicitly allowlisted in
  `scripts/lint/nav-allowlist.json` with a reason string; no duplicate nav
  entries; no nav entries without a file.
- **Redirect lint** (`scripts/lint/validate-redirects.ts`): a page you delete or
  move must gain a redirect **in the same PR** (see the
  [Redirects Guide](REDIRECTS_GUIDE.md)); no chained redirects; no redirects to
  non-existent targets; no duplicate redirect sources; no redirect source that
  shadows a live page; no internal links to non-existent paths.

Run them locally before pushing:

```bash
# from docs/public-docs (uses the tsx devDependency)
pnpm lint:nav
pnpm lint:redirects

# or from the monorepo root with bun (zero-dependency)
bun docs/public-docs/scripts/lint/validate-nav.ts
bun docs/public-docs/scripts/lint/validate-redirects.ts
```

Violations that pre-date the checks are grandfathered in
`scripts/lint/nav-allowlist.json` and `scripts/lint/redirect-lint-baseline.json`.
Those files only shrink: if your PR fixes a grandfathered violation, remove its
entry in the same PR (a stale entry fails the check). Never add a baseline entry
to silence a new violation — add the missing redirect or nav entry instead;
the allowlist is reserved for pages that are deliberately unlisted, with the
reason recorded.

## Pull Request Process

### Before Submitting
- Fix any reported issues
- Verify content accuracy
- Test all links and references
- Target the `develop` branch

### Submission Guidelines
1. Create a [new pull request](https://github.com/ethereum-optimism/optimism/compare)
2. Choose appropriate PR type or use blank template
3. Provide clear title and accurate description
4. Add labels

> **Important**: Add `flag:merge-pending-release` label if the PR content should only be released publicly in sync with a product release.

> **Tip**: Use "[Create draft pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request)" if your work is still in progress.

### Review Process
1. Assignment to Documentation team member
2. Technical review for accuracy
3. Quality and scope alignment check
4. Minimum 1 reviewer approval required
5. Reviewers will either approve, request changes, or close the pull request with comments
6. Automatic deployment after merge to [docs.optimism.io](https://docs.optimism.io)

## Code of Conduct
- Be respectful and inclusive
- Follow project guidelines
- Provide constructive feedback
- Maintain professional communication
- Report inappropriate behavior

## Additional Ways to Contribute
Even without direct code contributions, you can support us by:
- ⭐ Starring the project
- 🐦 Sharing on social media
- 📝 Mentioning us in your projects
- 🗣️ Spreading the word in your community

Thank you for contributing to Optimism Docs! 🎉
