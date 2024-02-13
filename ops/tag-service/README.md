# Tag Service
Tag Service is a Github action which builds new tags and applies them to services in the monorepo.
It accepts:
* Service name
* Bump Amount [major, minor, patch]
* Prerelease and Finalize-Prerelease (to add/remove `rc` versions)

It can be triggered from the Github Actions panel in the monorepo

# Tag Tool
Tag Tool is a minimal rewrite of the Tag Service to let operators prepare and commit tags from commandline
It accepts:
* Service name
* Bump Amount [major, minor, patch, prerelease, finalize-prerelease]

Tag Tool is meant to be run locally, and *does not* perform any write operations. Instead, it prints the git commands to console for the operator to use.

Additionally, a special service name "op-stack" is available, which will bump versions for `op-node`, `op-batcher` and `op-proposer` from the highest semver amongst them.

To run Tag Tool locally, the only dependency is `pip install semver`

