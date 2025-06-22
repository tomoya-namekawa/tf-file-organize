resource "google_service_account" "monitoring" {
  account_id   = "${local.service_name}-monitoring"
  display_name = "Monitoring Service Account"
}
resource "google_service_account" "service_account" {
  account_id   = "${local.service_name}-sa"
  display_name = "Service Account for ${local.service_name}"
  description  = "Service account for Cloud Run service ${local.service_name}"
}