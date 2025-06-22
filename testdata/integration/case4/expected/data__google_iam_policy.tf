

# Complex IAM policy with multiple statements

data "google_iam_policy" "complex_policy" {
  binding {
    role = "roles/run.invoker"
    members = [
      "allUsers",
      "serviceAccount:${google_service_account.service_account.email}"
    ]
    condition {
      title       = "Time-based access"
      description = "Only allow access during business hours"
      expression  = "request.time.getHours() >= 9 && request.time.getHours() <= 17"
    }
  }

  binding {
    role = "roles/run.developer"
    members = [
      "group:developers@${var.domain_name}",
      "user:admin@${var.domain_name}"
    ]
  }

  binding {
    role = "roles/logging.viewer"
    members = [
      "serviceAccount:${google_service_account.monitoring.email}"
    ]
    condition {
      title      = "Log viewer condition"
      expression = "resource.name.startsWith('projects/${var.project_id}/logs/cloudrun')"
    }
  }

  audit_config {
    service = "cloudrun.googleapis.com"
    audit_log_configs {
      log_type = "ADMIN_READ"
    }
    audit_log_configs {
      log_type         = "DATA_READ"
      exempted_members = ["user:admin@${var.domain_name}"]
    }
  }
}