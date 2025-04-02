# Rollup Boost Release Process

Guide for the release process of rollup-boost.

1. **Pre-Release Checks**

First of all, check all the CI tests and checks are running fine in the latest main branch.

Check out the latest main branch:

```bash
git checkout main
git pull origin main
```

Then run the following commands to check the code is working fine:

```bash
make lint
make test
make test-integration
git status # should show no changes

# Start rollup-boost with the example .env config
cargo run --

# Call the health endpoint
curl localhost:8081/healthz
```

2. **Release Candidate Process**

- Tag release candidate version:

```bash
git tag -s v0.5-rc1
git push origin --tags
```

- Test Docker image:

```bash
docker pull flashbots/rollup-boost:0.5rc1
```

3. **Testing & Validation**

- Test the docker image in internal testnets
  - Check no error logs in rollup-boost
  - Check no error logs in the builder or op-node
  - Check chain liveness is healthy and blocks are produced
  - Check the builder is landing blocks onchain by seeing if the builder transaction is included in the chain
  - Check metrics to see if there is any anomaly such as latency or blocks delivered
  - Use [contender](https://github.com/flashbots/contender) or other transaction spammer to check transactions are being included in the block
- Coordinate testing with external partners on testnets
  - Generally the same checklist as for internal testnets
  - Ask external partners if they observe any latency issues or error logs in rollup-boost
- Collect sign-offs from:
  - Op-Stack operators running rollup-boost
  - Code reviewers (check [recent contributors](https://github.com/flashbots/rollup-boost/graphs/contributors))

4. **Final Release Steps**

- Create final tags:

```bash
git tag -s v0.5
```

- Push all changes: `git push origin main --tags`
- Finalize GitHub release draft with:
  - Change log and new features
  - Breaking changes to the API or command line arguments
  - Compatibility with the latest OP Stack and relevant forks
  - Usage instructions (`rollup-boost --help`)

5. **Post-Release Tasks**

- Informing the community and relevant stakeholders, such as:
  - Internal team comms
  - External rollup-boost partners
  - Optimism Discord
