

# Google Cloud Run Service with complex nested blocks

resource "google_cloud_run_service" "complex_service" {
  name     = local.service_name
  location = var.region

  template {
    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale"  = "10"
        "run.googleapis.com/cpu-throttling" = "false"
      }
      labels = {
        environment = local.environment
        version     = var.build_number
      }
    }

    spec {
      service_account_name = google_service_account.service_account.email
      timeout_seconds      = 300

      containers {
        image = "gcr.io/${var.project_id}/app:${var.build_number}"

        resources {
          limits = {
            cpu    = "2000m"
            memory = "4Gi"
          }
          requests = {
            cpu    = "1000m"
            memory = "2Gi"
          }
        }

        ports {
          name           = "http1"
          container_port = 8080
          protocol       = "TCP"
        }

        env {
          name  = "PROJECT_ID"
          value = var.project_id
        }

        env {
          name  = "REGION"
          value = var.region
        }

        env {
          name  = "SERVICE_URL"
          value = "https://${local.full_domain}"
        }

        env {
          name  = "DATABASE_URL"
          value = "postgresql://user:pass@${google_sql_database_instance.main.private_ip_address}:5432/db"
        }

        env {
          name = "SECRET_KEY"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.app_secret.secret_id
              key  = "latest"
            }
          }
        }

        volume_mounts {
          name       = "config-volume"
          mount_path = "/etc/config"
        }

        volume_mounts {
          name       = "cache-volume"
          mount_path = "/tmp/cache"
        }

        startup_probe {
          http_get {
            path = "/health"
            port = 8080
            http_headers {
              name  = "Custom-Header"
              value = "startup"
            }
          }
          initial_delay_seconds = 10
          period_seconds        = 10
          timeout_seconds       = 5
          failure_threshold     = 3
        }

        liveness_probe {
          http_get {
            path = "/health"
            port = 8080
          }
          initial_delay_seconds = 30
          period_seconds        = 30
          timeout_seconds       = 10
        }
      }

      volumes {
        name = "config-volume"
        config_map {
          name = "app-config"
          items {
            key  = "app.yaml"
            path = "config.yaml"
          }
          items {
            key  = "logging.yaml"
            path = "logging.yaml"
          }
        }
      }

      volumes {
        name = "cache-volume"
        empty_dir {
          size_limit = "1Gi"
        }
      }
    }
  }

  traffic {
    percent         = 80
    latest_revision = false
    revision_name   = "${local.service_name}-v1"
  }

  traffic {
    percent         = 20
    latest_revision = true
  }

  autogenerate_revision_name = true

  lifecycle {
    ignore_changes = [
      template[0].metadata[0].annotations["client.knative.dev/user-image"]
    ]
  }
}