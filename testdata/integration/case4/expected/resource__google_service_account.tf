resource "google_service_account" "monitoring" {
  account_id   = "${local.service_name}-monitoring"
  display_name = "Monitoring Service Account"
}

resource "google_service_account" "service_account" {
  account_id   = "${local.service_name}-sa"
  description  = "Service account for Cloud Run service ${local.service_name}"
  display_name = "Service Account for ${local.service_name}"
}
