variable "REGISTRY" {
  default = "us-docker.pkg.dev"
}

variable "REPOSITORY" {
  default = "oplabs-tools-artifacts/internal-images"
}

variable "GIT_COMMIT" {
  default = "dev"
}

variable "GIT_DATE" {
  default = "0"
}

variable "GIT_VERSION" {
  default = "v0.0.0"
}

variable "IMAGE_TAGS" {
  default = "${GIT_COMMIT}" // split by ","
}

variable "PLATFORMS" {
  default = ""
}

target "ponder-interop" {
  dockerfile = "Dockerfile"
  context = "."
  args = {
    DOCKER_TARGET = "ponder-interop"
  }
  target = "ponder-interop"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/ponder-interop:${tag}"]
}

target "autorelayer-interop" {
  dockerfile = "Dockerfile"
  context = "."
  args = {
    DOCKER_TARGET = "autorelayer-interop"
  }
  target = "autorelayer-interop"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/autorelayer-interop:${tag}"]
}

target "sponsored-sender" {
  dockerfile = "Dockerfile"
  context = "."
  args = {
    DOCKER_TARGET = "sponsored-sender"
  }
  target = "sponsored-sender"
  platforms = split(",", PLATFORMS)
  tags = [for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/sponsored-sender:${tag}"]
}
