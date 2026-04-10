i'd like to automate our release process, which currently has a ton of tasks that are manual.

> Follow-up design doc: see `docs/oprm-design.md`.


let's devise a plan for a tool called "oprm" - OP release manager.

furthermore it is possible that the state for some of these tasks is modified, or they have been performed from another tool, so it should be possible to `retry` and `skip` certain tasks from the flow.

there should be a confirmation from the user on every step of the way, for every task to be performed.

we should store in a markdown file a log of all actions performed, and also who is the current release manager.

on startup check if `gh` is installed and if `git` is configured - that would be the release manager for the current schedule.

the process currently is as follows:

1. define which components to release from the OP stack. the tool should have support for:

- op-geth
- op-node
- op-batcher
- kona-node
- op-reth

2. determine which versions to release - in order to do that check the latest versions released from github and review git log if the components have changed (see ops/scripts/check-component-release.py from branch nonsense/check-component-release)

- it should be possible to define the version-to-be-released as a bump on `patch`, `minor`, or `major`, or manually defined by the release manager.

3. if `op-geth` is to be released start from it first, as it then needs to be referenced from `op-node` in its `go.mod`, which involves more commits to `develop`

4. verify that `develop` is exactly what we want to release, before cutting tags as a next step.

5. tag, push and create draft release notes on Github. ask for confirmation. detect if draft release notes already exist or not (maybe someone already created them in Github?), so make sure to explain if you are updating an existing entry or creating a new one.

- currently we do that with `./op tag <component> <patch-rc, ...>` from `op-workbench` - we might want to pull this code into our new project, or clone `op-workbench` as a submodule

6. wait and monitor for docker builds for the new version to complete in CircleCI

7. rollout (i.e. create PRs in k8s repo), get them merged, have the infrastructure updated, and if everything is well:

8. finalize the release (i.e. tag not just with rc, but also with the release versions)

let's skip steps 7 and 8 for now.

we should have a nice terminal UI that is easily modifiable.

ask any clarifying questions.
