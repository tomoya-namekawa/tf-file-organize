resource "google_kms_key_ring" "main" {
  name     = "${local.service_name}-keyring"
  location = var.region
}