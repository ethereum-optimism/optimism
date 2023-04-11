# The OP Stack Docs

[![Discord](https://img.shields.io/discord/667044843901681675.svg?color=768AD4&label=discord&logo=https%3A%2F%2Fdiscordapp.com%2Fassets%2F8c9701b98ad4372b58f13fd9f65f966e.svg)](https://discord.gg/optimism)
[![Twitter Follow](https://img.shields.io/twitter/follow/optimismPBC.svg?label=optimismPBC&style=social)](https://twitter.com/optimismPBC)

The OP Stack is an open, collectively maintained development stack for blockchain ecosystems.
This repository contains the source code for the [OP Stack Docs](https://stack.optimism.io).

## Development

### Serving docs locally

```sh
yarn dev
```

Then navigate to [http://localhost:8080](http://localhost:8080).
If that link doesn't work, double check the output of `yarn dev`.
You might already be serving something on port 8080 and the site may be on another port (e.g., 8081).

### Building docs for production

```sh
yarn build
```

You probably don't need to run this command, but now you know.

### Editing docs

Edit the markdown directly in [src/docs](./src/docs).

### Adding new docs

Add your markdown files to [src/docs](./src/docs).
You will also have to update [src/.vuepress/config.js](./src/.vuepress/config.js) if you want these docs to show up in the sidebar.

### Updating the theme

We currently use an ejected version of [vuepress-theme-hope](https://vuepress-theme-hope.github.io/).
Since the version we use was ejected from the original theme, you'll see a bunch of compiled JavaScript files instead of the original TypeScript files.
There's not much we can do about that right now, so you'll just need to make do and edit the raw JS if you need to make theme adjustments.
We're planning to move away from VuePress relatively soon anyway so we won't be fixing this.
