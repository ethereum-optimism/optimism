# Releases

This is a concise guide for cutting a release for `kona` crates.
> [!TIP]
>
> Ensure you have permission to update any affected crates before cutting a release.


### cargo-release

Ensure [cargo-release][cargo-release] is installed using cargo's `install` command.

```
$ cargo install cargo-release
```

### Dry Run

> [!TIP]
>
> Ensure that you have trunk (the `main` branch) checked out and up to date.

Let's say we want to release the `kona-protocol` crate.
Execute the following command to perform a _dry run_ patch release.

```
$ cargo release patch --package kona-protocol --no-push
```

This will update the _patch_ version of the crate. (e.g. `0.1.0` -> `0.1.1`).

To update minor and major versions, just specify `minor` or `major` in place of `patch`.

If this command executes without any errors, proceed to executing the release.

### Cutting the Release

> [!IMPORTANT]
>
> Executing the release command may take time depending on your machine and
> how quickly it can compile the crate. Be prepared to let this run for some time.

Append the `--execute` argument to the cargo release command to execute the dry run above.

```
$ cargo release patch --package kona-protocol --no-push --execute
```

The `kona-protocol` crate will be published.
Once this is done be sure to push the artifacts in the next step!

### Committing Artifacts

After the release command completes, it will automatically commit artifacts to the current
branch - `main`.

Since we don't want to push to the `main` branch, we need to do a few things.

Reset the git commit so changes are not committed like so.

```
$ git reset HEAD^
```

Running `git status` should show unstaged changes to the `CHANGELOG.md`
as well as cargo manifest TOMLs.

Now, checkout a new branch, commit, and push the artifacts to the new branch.

```
$ git checkout -b release/kona-protocol/0.1.1
$ git add .
$ git commit -m "release(kona-protocol): 0.1.1"
$ git push
```

Open a PR and you're all set, the release is complete!


<!-- Hyperlinks -->

[cargo-release]: https://github.com/crate-ci/cargo-release
