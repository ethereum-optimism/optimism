variable "REGISTRY" {
  default = "us-docker.pkg.dev"
}

variable "REPOSITORY" {
  default = "oplabs-tools-artifacts/images"
}

variable "GIT_COMMIT" {
  default = "dev"
}

variable "GIT_DATE" {
  default = "0"
}

// The default version to embed in the built images.
// During CI release builds this is set to <<pipeline.git.tag>>
variable "GIT_VERSION" {
  default = "v0.0.0"
}

variable "IMAGE_TAGS" {
  default = "${GIT_COMMIT}" // split by ","
}

variable "PLATFORMS" {
  // You can override this as "linux/amd64,linux/arm64".
  // Only specify a single platform when `--load` ing into docker.
  // Multi-platform is supported when outputting to disk or pushing to a registry.
  // Multi-platform builds can be tested locally with:  --set="*.output=type=image,push=false"
  default = ""
}

// Each of the services can have a customized version, but defaults to the global specified version.
variable "OP_NODE_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_BATCHER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_PROPOSER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_CHALLENGER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_DISPUTE_MON_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_SUPERNODE_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_SILHOUETTE_EL_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_INTEROP_FILTER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_TEST_SEQUENCER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "CANNON_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_CONDUCTOR_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_DEPLOYER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_DRIPPER_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_FAUCET_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_INTEROP_MON_VERSION" {
  default = "${GIT_VERSION}"
}


target "op-node" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_NODE_VERSION = "${OP_NODE_VERSION}"
  }
  target = "op-node-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-node:${tag}"]
}

target "op-batcher" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_BATCHER_VERSION = "${OP_BATCHER_VERSION}"
  }
  target = "op-batcher-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-batcher:${tag}"]
}

target "op-proposer" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_PROPOSER_VERSION = "${OP_PROPOSER_VERSION}"
  }
  target = "op-proposer-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-proposer:${tag}"]
}

target "op-challenger" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_CHALLENGER_VERSION = "${OP_CHALLENGER_VERSION}"
  }
  target = "op-challenger-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-challenger:${tag}"]
}

target "op-dispute-mon" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_DISPUTE_MON_VERSION = "${OP_DISPUTE_MON_VERSION}"
  }
  target = "op-dispute-mon-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-dispute-mon:${tag}"]
}

target "op-conductor" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_CONDUCTOR_VERSION = "${OP_CONDUCTOR_VERSION}"
  }
  target = "op-conductor-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-conductor:${tag}"]
}

target "da-server" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
  }
  target = "da-server-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/da-server:${tag}"]
}

target "op-supernode" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_SUPERNODE_VERSION = "${OP_SUPERNODE_VERSION}"
  }
  target = "op-supernode-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-supernode:${tag}"]
}

target "op-silhouette-el" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_SILHOUETTE_EL_VERSION = "${OP_SILHOUETTE_EL_VERSION}"
  }
  target = "op-silhouette-el-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-silhouette-el:${tag}"]
}

target "op-interop-filter" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_INTEROP_FILTER_VERSION = "${OP_INTEROP_FILTER_VERSION}"
  }
  target = "op-interop-filter-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-interop-filter:${tag}"]
}

target "op-test-sequencer" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_TEST_SEQUENCER_VERSION = "${OP_TEST_SEQUENCER_VERSION}"
  }
  target = "op-test-sequencer-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-test-sequencer:${tag}"]
}

target "cannon" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    CANNON_VERSION = "${CANNON_VERSION}"
  }
  target = "cannon-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/cannon:${tag}"]
}

target "op-deployer" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_DEPLOYER_VERSION = "${OP_DEPLOYER_VERSION}"
  }
  target = "op-deployer-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-deployer:${tag}"]
}

target "op-dripper" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_DRIPPER_VERSION = "${OP_DRIPPER_VERSION}"
  }
  target = "op-dripper-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-dripper:${tag}"]
}

target "op-faucet" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_FAUCET_VERSION = "${OP_FAUCET_VERSION}"
  }
  target = "op-faucet-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-faucet:${tag}"]
}

target "op-interop-mon" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_INTEROP_MON_VERSION = "${OP_INTEROP_MON_VERSION}"
  }
  target = "op-interop-mon-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-interop-mon:${tag}"]
}

// Rust-based images

target "kona-node" {
  dockerfile = "kona/docker/apps/kona_app_generic.dockerfile"
  context = "rust"
  contexts = {
    nuts-bundles = "op-core/nuts/bundles"
  }
  args = {
    REPO_LOCATION = "local"
    BIN_TARGET = "kona-node"
    BUILD_PROFILE = "release"
    GIT_VERSION = "${GIT_VERSION}"
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/kona-node:${tag}"]
}

target "kona-host" {
  dockerfile = "kona/docker/apps/kona_app_generic.dockerfile"
  context = "rust"
  contexts = {
    nuts-bundles = "op-core/nuts/bundles"
  }
  args = {
    REPO_LOCATION = "local"
    BIN_TARGET = "kona-host"
    BUILD_PROFILE = "release"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/kona-host:${tag}"]
}

target "kona-client" {
  dockerfile = "kona/docker/apps/kona_app_generic.dockerfile"
  context = "rust"
  contexts = {
    nuts-bundles = "op-core/nuts/bundles"
  }
  args = {
    REPO_LOCATION = "local"
    BIN_TARGET = "kona-client"
    BUILD_PROFILE = "release"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/kona-client:${tag}"]
}

target "kona-sp1-proposer" {
  dockerfile = "kona/docker/apps/kona_app_generic.dockerfile"
  context = "rust"
  contexts = {
    nuts-bundles = "op-core/nuts/bundles"
    contracts-bedrock-abis = "packages/contracts-bedrock/snapshots/abi"
  }
  args = {
    REPO_LOCATION = "local"
    BUILDER_VARIANT = "contract-abis"
    BIN_TARGET = "kona-sp1-proposer"
    BUILD_PROFILE = "release"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/kona-sp1-proposer:${tag}"]
}

target "op-reth" {
  dockerfile = "op-reth/DockerfileOp"
  context = "rust"
  # The chainspec build.rs reads the superchain-registry submodule (at the repo
  # root, outside the "rust" context); expose it as a named context so the
  # Dockerfile can COPY the subset it needs.
  contexts = {
    superchain-registry = "superchain-registry"
  }
  args = {
    BUILD_PROFILE = "maxperf"
    GIT_VERSION = "${GIT_VERSION}"
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-reth:${tag}"]
}

target "cannon-builder" {
  dockerfile = "cannon.dockerfile"
  context = "rust/kona/docker/cannon"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/cannon-builder:${tag}"]
}
