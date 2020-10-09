data "terraform_remote_state" "vpn" {
  backend = "local"

  config = {
    path = "../terraform-vpn/terraform.tfstate"
  }
}
