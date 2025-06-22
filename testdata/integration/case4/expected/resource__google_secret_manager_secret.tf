resource "google_secret_manager_secret" "app_secret" {
  labels    = local.common_tags
  secret_id = "${local.service_name}-secret"
}
