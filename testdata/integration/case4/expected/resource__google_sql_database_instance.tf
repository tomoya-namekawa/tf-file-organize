resource "google_sql_database_instance" "main" {
  name             = "${local.service_name}-db"
  database_version = "POSTGRES_13"
  region           = var.region

  settings {
    tier              = "db-custom-2-8192"
    availability_type = "REGIONAL"
    disk_type         = "PD_SSD"
    disk_size         = 100
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 30
        retention_unit   = "COUNT"
      }
    }

    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.main.id
      authorized_networks {
        name  = "internal"
        value = "10.0.0.0/8"
      }
      authorized_networks {
        name  = "office"
        value = "203.0.113.0/24"
      }
    }

    database_flags {
      name  = "shared_preload_libraries"
      value = "pg_stat_statements"
    }

    database_flags {
      name  = "log_min_duration_statement"
      value = "1000"
    }

    maintenance_window {
      day          = 7
      hour         = 3
      update_track = "stable"
    }

    insights_config {
      query_insights_enabled  = true
      record_application_tags = true
      record_client_address   = true
    }
  }

  deletion_protection = true

  lifecycle {
    prevent_destroy = true
    ignore_changes = [
      settings[0].disk_size
    ]
  }
}