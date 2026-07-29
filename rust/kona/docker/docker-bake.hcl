////////////////////////////////////////////////////////////////
//                          Globals                           //
////////////////////////////////////////////////////////////////

variable "REGISTRY" {
  default = "ghcr.io"
}

variable "REPOSITORY" {
  default = "ethereum-optimism/kona"
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
  dockerfile = "kona/docker/apps/kona_app_generic.dockerfile"
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

// Rust build environment for bare-metal MIPS64r1 (Cannon FPVM ISA).
// Contains only apt-level MIPS64 cross-compilation packages.
// Rust, Go, mise, and just are installed on top from pinned version sources.
target "cannon-builder" {
  inherits = ["docker-metadata-action"]
  context = "kona/docker/cannon"
  dockerfile = "cannon.dockerfile"
  args = {
    HOST_UID = "${HOST_UID}"
    HOST_GID = "${HOST_GID}"
  }
  platforms = split(",", PLATFORMS)
}
