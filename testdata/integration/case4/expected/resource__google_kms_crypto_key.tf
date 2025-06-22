resource "google_kms_crypto_key" "secret_key" {
  key_ring = google_kms_key_ring.main.id
  name     = "${local.service_name}-secret-key"
  lifecycle {
    prevent_destroy = true
  }
}
