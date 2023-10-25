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

variable "GIT_VERSION" {
  default = "docker"  // original default as set in proxyd file, not used by full go stack, yet
}

variable "IMAGE_TAGS" {
  default = "${GIT_COMMIT}" // split by ","
}

variable "PLATFORMS" {
  // You can override this as "linux/amd64,linux/arm64".
  // Only a specify a single platform when `--load` ing into docker.
  // Multi-platform is supported when outputting to disk or pushing to a registry.
  // Multi-platform builds can be tested locally with:  --set="*.output=type=image,push=false"
  default = "linux/amd64"
}

target "op-stack-go" {
  dockerfile = "ops/docker/op-stack-go/Dockerfile"
  context = "."
  args = {
    GIT_COMMIT = "${GIT_COMMIT}"
    GIT_DATE = "${GIT_DATE}"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-stack-go:${tag}"]
}

target "op-node" {
  dockerfile = "Dockerfile"
  context = "./op-node"
  args = {
    OP_STACK_GO_BUILDER = "op-stack-go"
  }
  contexts = {
    op-stack-go: "target:op-stack-go"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-node:${tag}"]
}

target "op-batcher" {
  dockerfile = "Dockerfile"
  context = "./op-batcher"
  args = {
    OP_STACK_GO_BUILDER = "op-stack-go"
  }
  contexts = {
    op-stack-go: "target:op-stack-go"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-batcher:${tag}"]
}

target "op-proposer" {
  dockerfile = "Dockerfile"
  context = "./op-proposer"
  args = {
    OP_STACK_GO_BUILDER = "op-stack-go"
  }
  contexts = {
    op-stack-go: "target:op-stack-go"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-proposer:${tag}"]
}

target "op-challenger" {
  dockerfile = "Dockerfile"
  context = "./op-challenger"
  args = {
    OP_STACK_GO_BUILDER = "op-stack-go"
  }
  contexts = {
    op-stack-go: "target:op-stack-go"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-challenger:${tag}"]
}

target "op-heartbeat" {
  dockerfile = "Dockerfile"
  context = "./op-heartbeat"
  args = {
    OP_STACK_GO_BUILDER = "op-stack-go"
  }
  contexts = {
    op-stack-go: "target:op-stack-go"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-heartbeat:${tag}"]
}

target "op-program" {
  dockerfile = "Dockerfile"
  context = "./op-program"
  args = {
    OP_STACK_GO_BUILDER = "op-stack-go"
  }
  contexts = {
    op-stack-go: "target:op-stack-go"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-program:${tag}"]
}

target "proxyd" {
  dockerfile = "Dockerfile"
  context = "./proxyd"
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

target "ufm-metamask" {
  dockerfile = "Dockerfile"
  context = "./ufm-test-services/metamask"
  args = {
    // proxyd dockerfile has no _ in the args
    GITCOMMIT = "${GIT_COMMIT}"
    GITDATE = "${GIT_DATE}"
    GITVERSION = "${GIT_VERSION}"
  }
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/ufm-metamask:${tag}"]
}

type "chain-mon" {
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

type "ci-builder" {
  dockerfile = "Dockerfile"
  context = "ops/docker/ci-builder"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/ci-builder:${tag}"]
}


