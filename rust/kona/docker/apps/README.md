# `docker-apps`

This directory contains a dockerfile for building any binary in the `kona` repository. It supports building both the
local repository as well as a remote revision.

## Building

To build an image for any binary within `kona` locally, use the `justfile` located in this directory:

```sh
# Build an application image from the local repository
just build-local <bin_name> [image_tag (default: 'kona:local')]

# Build an application image from a remote revision
just build-remote <bin_name> <git_tag> [image_tag (default: 'kona:local')]
```

### Configuration

#### Image Platform

By default, the `justfile` directives will build the image only for the host machine's platform. This can be overridden
by setting the `PLATFORMS` environment variable to a comma-separated list, like so:

```sh
export PLATFORMS="linux/amd64,linux/arm64,linux/aarch64"
```

#### Cargo Build Profile

By default, the `release` profile will be used for the application's build. This can be overridden by setting the
`BUILD_PROFILE` environment variable to the desired profile, like so:

```sh
export BUILD_PROFILE="debug"
```

## Publishing App Images

The `generic` target in the [`docker-bake.hcl`](../docker-bake.hcl) supports publishing application binary images for
any binary in the `kona` repository. Optionally, if a custom target is desired, it can be overridden in the
[`docker-bake.hcl`](../docker-bake.hcl) like so:

```hcl
target "<bin-name>" {
  inherits = ["docker-metadata-action"]
  context = "."
  dockerfile = "docker/apps/<custom>.dockerfile"
  args = {
    REPO_LOCATION = "${REPO_LOCATION}"
    REPOSITORY = "${REPOSITORY}"
    TAG = "${GIT_REF_NAME}"
    BIN_TARGET = "<bin-name>"
    BUILD_PROFILE = "${BUILD_PROFILE}"
  }
  platforms = split(",", PLATFORMS)
}
```

The [docker release workflow](../../.github/workflows/docker.yaml) will **first** check if a target is available,
using that target for the docker build. If the workflow can't find the target, it will fallback to the "generic"
target specified in the [`docker-bake.hcl`](../docker-bake.hcl).

To cut a release for a generic binary, or an overridden target for that matter, follow the guidelines specified
in the ["cutting a release"](../README.md#cutting-a-release-for-maintainers--forks) section. This workflow allows
you to trigger a release just by pushing a tag to kona, for any binary. No code changes needed :)
