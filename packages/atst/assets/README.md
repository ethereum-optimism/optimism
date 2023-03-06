# Assets

## preview.gif

A gif preview of using the cli

## preview.tape

The script to record the preview.gif with [vhs](https://github.com/charmbracelet/vhs)

To execute:

1. [Download vhs](https://github.com/charmbracelet/vhs)

2. Install the local version of atst

```bash
npm uninstall @eth-optimism/atst -g && npm i . -g && atst --version
```

3. Start anvil

```bash
anvil --fork-url https://mainnet.optimism.io
```

4. Record tape vhs < assets/preview.tape

```bash
vhs < assets/preview.tape
```

5. The tape will be outputted to `assets/preview.gif`
