resource "google_sql_database_instance" "main" {
  database_version    = "POSTGRES_13"
  deletion_protection = true
  name                = "${local.service_name}-db"
  region              = var.region
  settings {
    availability_type = "REGIONAL"
    disk_autoresize   = true
    disk_size         = 100
    disk_type         = "PD_SSD"
    tier              = "db-custom-2-8192"
  }
  lifecycle {
    ignore_changes  = [settings[0].disk_size]
    prevent_destroy = true
  }
}
