resource "google_cloud_run_service" "complex_service" {
  autogenerate_revision_name = true
  location                   = var.region
  name                       = local.service_name
  template {
    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale"  = "10"
        "run.googleapis.com/cpu-throttling" = "false"
      }
      labels = {}
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
          container_port = 8080
          name           = "http1"
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
        }
      }
    }
  }
  traffic {
    latest_revision = false
    percent         = 80
    revision_name   = "${local.service_name}-v1"
  }
  traffic {
    latest_revision = true
    percent         = 20
  }
  lifecycle {
    ignore_changes = [template[0].metadata[0].annotations["client.knative.dev/user-image"]]
  }
}
