resource "google_cloud_run_domain_mapping" "domain" {
  location = var.region
  name     = "${var.subdomain}.${var.domain_name}"

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = google_cloud_run_service.hooks.name
  }
}