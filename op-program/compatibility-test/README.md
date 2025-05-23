# Compatibility Test Baseline

These baselines are used as part of the `run-vm-compat` task to check op-program for any unsupported
opcodes or syscalls.

## Simplifying `vm-compat` Output

When the analysis job fails it prints JSON output for all new findings. To format these nicely and remove the `line`,
`file` and `absPath` fields to match the existing baseline, use `jq`:

```shell
pbpaste | jq 'walk(if type == "object" and has("line") then del(.line) else . end | if type == "object" and has("absPath") then del(.absPath) else . end | if type == "object" and has("file") then del(.file) else . end)' | pbcopy
```

`pbpaste` and `pbcopy` are MacOS specific commands. They make it easy to copy the output from the CI results, run that
command and the formatted result is left on the clipboard ready to be pasted in. The `jq` command itself will work fine
on Linux.

Since these fields are ignored by `vm-compat` (to reduce false positives when line numbers change), it simplifies the
diff substantially to exclude them from the committed baseline file.
