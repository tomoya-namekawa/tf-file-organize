

# KMS Key

resource "google_kms_crypto_key" "secret_key" {
  name     = "${local.service_name}-secret-key"
  key_ring = google_kms_key_ring.main.id

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = "SOFTWARE"
  }

  lifecycle {
    prevent_destroy = true
  }
}