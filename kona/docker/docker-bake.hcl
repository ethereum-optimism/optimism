////////////////////////////////////////////////////////////////
//                          Globals                           //
////////////////////////////////////////////////////////////////

variable "REGISTRY" {
  default = "ghcr.io"
}

variable "REPOSITORY" {
  default = "op-rs/kona"
}

variable "DEFAULT_TAG" {
  default = "kona:local"
  description = "The tag to use for the built image."
}

variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
  description = "The platforms to build the image for, separated by commas."
}

variable "GIT_REF_NAME" {
  default = "main"
  description = "The git reference name. This is typically the branch name, commit hash, or tag."
}

// Special target: https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {
  description = "Special target used with `docker/metadata-action`"
  tags = ["${DEFAULT_TAG}"]
}

////////////////////////////////////////////////////////////////
//                         App Images                         //
////////////////////////////////////////////////////////////////

variable "REPO_LOCATION" {
  default = "remote"
  description = "The location of the repository to build in the kona-app-generic target. Valid options: local (uses local repo, ignores `GIT_REF_NAME`), remote (clones `kona`, checks out `GIT_REF_NAME`)"
}

variable "BIN_TARGET" {
  default = "kona-host"
  description = "The binary target to build in the kona-app-generic target."
}

variable "BUILD_PROFILE" {
  default = "release-perf"
  description = "The cargo build profile to use when building the binary in the kona-app-generic target."
}

target "generic" {
  description = "Generic kona app image"
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

variable "ASTERISC_TAG" {
  // The tag of `asterisc` to use in the `kona-asterisc-prestate` target.
  //
  // You can override this if you'd like to use a different tag to generate the prestate.
  // https://github.com/ethereum-optimism/asterisc/releases
  default = "v1.3.0"
  description = "The tag of asterisc to use in the kona-asterisc-prestate target."
}

variable "CANNON_TAG" {
  // The tag of `cannon` to use in the `kona-cannon-prestate` target.
  //
  // You can override this if you'd like to use a different tag to generate the prestate.
  // https://github.com/ethereum-optimism/optimism/releases
  default = "cannon/v1.5.0-alpha.1"
  description = "The tag of cannon to use in the kona-cannon-prestate target."
}

variable "CLIENT_BIN" {
  // The `kona-client` binary to use in the `kona-{asterisc/cannon}-prestate` targets.
  //
  // You can override this if you'd like to use a different `kona-client` binary to generate
  // the prestate.
  //
  // Valid options:
  // - `kona` (single-chain)
  // - `kona-int` (interop)
  default = "kona"
  description = "The kona-client binary to use in the proof prestate targets. Valid options: kona, kona-int"
}

variable "KONA_CUSTOM_CONFIGS" {
  // Used to build a kona prestate using custom chain configurations
  default = "false"
  description = "Enables custom chain configurations to be built into kona artifacts"
}

variable "CUSTOM_CONFIGS_CONTEXT" {
  // The build context for custom chain configurations to add to the prestate build
  default = ""
  description = "The build context for custom chain configurations to add to the prestate build"
}


target "asterisc-builder" {
  description = "Rust build environment for bare-metal RISC-V 64-bit IMA (Asterisc FPVM ISA)"
  inherits = ["docker-metadata-action"]
  context = "docker/asterisc"
  dockerfile = "asterisc.dockerfile"
  platforms = split(",", PLATFORMS)
}

target "cannon-builder" {
  description = "Rust build environment for bare-metal MIPS64r1 (Cannon FPVM ISA)"
  inherits = ["docker-metadata-action"]
  context = "docker/cannon"
  dockerfile = "cannon.dockerfile"
  platforms = split(",", PLATFORMS)
}

target "kona-asterisc-prestate" {
  description = "Prestate builder for kona-client with Asterisc FPVM"
  inherits = ["docker-metadata-action"]
  context = "."
  dockerfile = "docker/fpvm-prestates/asterisc-repro.dockerfile"
  args = {
    CLIENT_BIN = "${CLIENT_BIN}"
    CLIENT_TAG = "${GIT_REF_NAME}"
    ASTERISC_TAG = "${ASTERISC_TAG}"
  }
  # Only build on linux/amd64 for reproducibility.
  platforms = ["linux/amd64"]
}

target "kona-cannon-prestate" {
  description = "Prestate builder for kona-client with Cannon FPVM"
  inherits = ["docker-metadata-action"]
  context = "."
  dockerfile = "docker/fpvm-prestates/cannon-repro.dockerfile"
  contexts = {
    custom_configs = "${CUSTOM_CONFIGS_CONTEXT}"
  }
  args = {
    CLIENT_BIN = "${CLIENT_BIN}"
    CLIENT_TAG = "${GIT_REF_NAME}"
    CANNON_TAG = "${CANNON_TAG}"
    KONA_CUSTOM_CONFIGS = "${KONA_CUSTOM_CONFIGS}"
  }
  # Only build on linux/amd64 for a single source of reproducibility.
  platforms = ["linux/amd64"]
}
