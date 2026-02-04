////////////////////////////////////////////////////////////////
//                          Globals                           //
////////////////////////////////////////////////////////////////

variable "REGISTRY" {
  default = "ghcr.io"
}

variable "REPOSITORY" {
  default = "op-rs/kona"
}

// The tag to use for the built image.
variable "DEFAULT_TAG" {
  default = "kona:local"
}

// The platforms to build the image for, separated by commas.
variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
}

// The git reference name. This is typically the branch name, commit hash, or tag.
variable "GIT_REF_NAME" {
  default = "main"
}

// The UID of the host user for volume permissions.
variable "HOST_UID" {
  default = "1000"
}

// The GID of the host user for volume permissions.
variable "HOST_GID" {
  default = "1000"
}

// Special target: https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {
  tags = ["${DEFAULT_TAG}"]
}

////////////////////////////////////////////////////////////////
//                         App Images                         //
////////////////////////////////////////////////////////////////

// The location of the repository to build in the kona-app-generic target. Valid options: local (uses local repo, ignores `GIT_REF_NAME`), remote (clones `kona`, checks out `GIT_REF_NAME`)
variable "REPO_LOCATION" {
  default = "remote"
}

// The binary target to build in the kona-app-generic target.
variable "BIN_TARGET" {
  default = "kona-host"
}

// The cargo build profile to use when building the binary in the kona-app-generic target.
variable "BUILD_PROFILE" {
  default = "release-perf"
}

// Generic kona app image
target "generic" {
  inherits = ["docker-metadata-action"]
  context = "."
  dockerfile = "docker/apps/kona_app_generic.dockerfile"
  args = {
    REPO_LOCATION = "${REPO_LOCATION}"
    REPOSITORY = "${REPOSITORY}"
    TAG = "${GIT_REF_NAME}"
    BIN_TARGET = "${BIN_TARGET}"
    BUILD_PROFILE = "${BUILD_PROFILE}"
  }
  platforms = split(",", PLATFORMS)
}

////////////////////////////////////////////////////////////////
//                        Proof Images                        //
////////////////////////////////////////////////////////////////

// The tag of `cannon` to use in the `kona-cannon-prestate` target.
//
// You can override this if you'd like to use a different tag to generate the prestate.
// https://github.com/ethereum-optimism/optimism/releases
variable "CANNON_TAG" {
  default = "cannon/v1.5.0-alpha.1"
}

// The `kona-client` binary to use in the `kona-cannon-prestate` target.
//
// You can override this if you'd like to use a different `kona-client` binary to generate
// the prestate.
//
// Valid options:
// - `kona` (single-chain)
// - `kona-int` (interop)
variable "CLIENT_BIN" {
  default = "kona"

}

// Enables custom chain configurations to be built into kona artifacts
variable "KONA_CUSTOM_CONFIGS" {
  default = "false"

}

// The build context for custom chain configurations to add to the prestate build
variable "CUSTOM_CONFIGS_CONTEXT" {
  default = ""

}


// Rust build environment for bare-metal MIPS64r1 (Cannon FPVM ISA)
target "cannon-builder" {
  inherits = ["docker-metadata-action"]
  context = "docker/cannon"
  dockerfile = "cannon.dockerfile"
  args = {
    HOST_UID = "${HOST_UID}"
    HOST_GID = "${HOST_GID}"
  }
  platforms = split(",", PLATFORMS)
}

// Prestate builder for kona-client with Cannon FPVM
target "kona-cannon-prestate" {
  inherits = ["docker-metadata-action"]
  context = "."
  dockerfile = "docker/fpvm-prestates/cannon-repro.dockerfile"
  contexts = {
    custom_configs = "${CUSTOM_CONFIGS_CONTEXT}"
  }
  args = {
    CLIENT_BIN = "${CLIENT_BIN}"
    CANNON_TAG = "${CANNON_TAG}"
    KONA_CUSTOM_CONFIGS = "${KONA_CUSTOM_CONFIGS}"
  }
  # Only build on linux/amd64 for a single source of reproducibility.
  platforms = ["linux/amd64"]
}
