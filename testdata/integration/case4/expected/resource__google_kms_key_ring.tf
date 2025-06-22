resource "google_kms_key_ring" "main" {
  location = var.region
  name     = "${local.service_name}-keyring"
}
