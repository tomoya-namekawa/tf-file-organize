data "google_iam_policy" "complex_policy" {
  binding {
    members = ["", ""]
    role    = "roles/run.invoker"
    condition {
      description = "Only allow access during business hours"
      expression  = "request.time.getHours() >= 9 && request.time.getHours() <= 17"
      title       = "Time-based access"
    }
  }
  binding {
    members = ["", ""]
    role    = "roles/run.developer"
  }
  binding {
    members = [""]
    role    = "roles/logging.viewer"
    condition {
      expression = "resource.name.startsWith('projects/${var.project_id}/logs/cloudrun')"
      title      = "Log viewer condition"
    }
  }
}
