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
  // Only a specify a single platform when `--load` ing into docker.
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

variable "OP_HEARTBEAT_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_PROGRAM_VERSION" {
  default = "${GIT_VERSION}"
}

variable "CANNON_VERSION" {
  default = "${GIT_VERSION}"
}

variable "OP_CONDUCTOR_VERSION" {
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

target "op-heartbeat" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_HEARTBEAT_VERSION = "${OP_HEARTBEAT_VERSION}"
  }
  target = "op-heartbeat-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-heartbeat:${tag}"]
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

target "op-program" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
    OP_PROGRAM_VERSION = "${OP_PROGRAM_VERSION}"
  }
  target = "op-program-target"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-program:${tag}"]
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

target "proxyd" {
  dockerfile = "./proxyd/Dockerfile"
  context = "./"
  args = {
    // proxyd dockerfile has no _ in the args
    GITCOMMIT = "${GIT_COMMIT}"
    GITDATE = "${GIT_DATE}"
    GITVERSION = "${GIT_VERSION}"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/proxyd:${tag}"]
}

target "indexer" {
  dockerfile = "./indexer/Dockerfile"
  context = "./"
  args = {
    // proxyd dockerfile has no _ in the args
    GITCOMMIT = "${GIT_COMMIT}"
    GITDATE = "${GIT_DATE}"
    GITVERSION = "${GIT_VERSION}"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/indexer:${tag}"]
}

target "chain-mon" {
  dockerfile = "./ops/docker/Dockerfile.packages"
  context = "."
  args = {
    // proxyd dockerfile has no _ in the args
    GITCOMMIT = "${GIT_COMMIT}"
    GITDATE = "${GIT_DATE}"
    GITVERSION = "${GIT_VERSION}"
  }
  // this is a multi-stage build, where each stage is a possible output target, but wd-mon is all we publish
  target = "wd-mon"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/chain-mon:${tag}"]
}

target "ci-builder" {
  dockerfile = "./ops/docker/ci-builder/Dockerfile"
  context = "."
  platforms = split(",", PLATFORMS)
  target="base-builder"
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/ci-builder:${tag}"]
}

target "ci-builder-rust" {
  dockerfile = "./ops/docker/ci-builder/Dockerfile"
  context = "."
  platforms = split(",", PLATFORMS)
  target="rust-builder"
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/ci-builder-rust:${tag}"]
}

target "contracts-bedrock" {
  dockerfile = "./ops/docker/Dockerfile.packages"
  context = "."
  target = "contracts-bedrock"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/contracts-bedrock:${tag}"]
}
