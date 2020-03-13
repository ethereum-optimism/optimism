provider "google" {
  project = "${var.project}"
  region  = "us-central1"

  credentials = "${file("vault-helm-dev-creds.json")}"
}

resource "random_id" "suffix" {
  byte_length = 4
}

data "google_container_engine_versions" "main" {
  location = "${var.zone}"
  version_prefix = "1.15."
}

data "google_service_account" "gcpapi" {
  account_id = "${var.gcp_service_account}"
}

resource "google_kms_key_ring" "keyring" {
  name     = "vault-helm-unseal-kr"
  location = "global"
}

resource "google_kms_crypto_key" "vault-helm-unseal-key" {
  name            = "vault-helm-unseal-key"
  key_ring        = "${google_kms_key_ring.keyring.self_link}"

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_container_cluster" "cluster" {
  name               = "vault-helm-dev-${random_id.suffix.dec}"
  project            = "${var.project}"
  enable_legacy_abac = true
  initial_node_count = 3
  location           = "${var.zone}"
  min_master_version = "${data.google_container_engine_versions.main.latest_master_version}"
  node_version       = "${data.google_container_engine_versions.main.latest_node_version}"

  node_config {
    #service account for nodes to use
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_write",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    service_account = "${data.google_service_account.gcpapi.email}"
  }
}

resource "null_resource" "kubectl" {
  count = "${var.init_cli ? 1 : 0 }"

  triggers = {
    cluster = "${google_container_cluster.cluster.id}"
  }

  # On creation, we want to setup the kubectl credentials. The easiest way
  # to do this is to shell out to gcloud.
  provisioner "local-exec" {
    command = "gcloud container clusters get-credentials --zone=${var.zone} ${google_container_cluster.cluster.name}"
  }

  # On destroy we want to try to clean up the kubectl credentials. This
  # might fail if the credentials are already cleaned up or something so we
  # want this to continue on failure. Generally, this works just fine since
  # it only operates on local data.
  provisioner "local-exec" {
    when       = "destroy"
    on_failure = "continue"
    command    = "kubectl config get-clusters | grep ${google_container_cluster.cluster.name} | xargs -n1 kubectl config delete-cluster"
  }

  provisioner "local-exec" {
    when       = "destroy"
    on_failure = "continue"
    command    = "kubectl config get-contexts | grep ${google_container_cluster.cluster.name} | xargs -n1 kubectl config delete-context"
  }
}
