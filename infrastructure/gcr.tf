/*
 * Google Container Registry: https://www.terraform.io/docs/providers/google/r/container_registry.html
 * Private container registry to push and pull proprietary pod images
 */
resource "google_container_registry" "registry" {
  // No arguments are needed, `project` is inherited from provider
}
