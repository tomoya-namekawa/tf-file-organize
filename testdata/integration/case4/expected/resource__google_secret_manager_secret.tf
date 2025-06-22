

# Secret Manager secret

resource "google_secret_manager_secret" "app_secret" {
  secret_id = "${local.service_name}-secret"

  replication {
    user_managed {
      replicas {
        location = var.region
        customer_managed_encryption {
          kms_key_name = google_kms_crypto_key.secret_key.id
        }
      }
      replicas {
        location = "us-east1"
      }
    }
  }

  labels = local.common_tags
}