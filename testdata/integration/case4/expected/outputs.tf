output "database_connection" {
  description = "Database connection string"
  value       = "postgresql://user:password@${google_sql_database_instance.main.private_ip_address}:5432/database"
  sensitive   = true
}
output "service_internal_url" {
  description = "Internal URL of the service"
  value       = "https://${google_cloud_run_service.complex_service.status[0].url}"
}
output "service_url" {
  description = "URL of the Cloud Run service"
  value       = "https://${local.full_domain}"
}