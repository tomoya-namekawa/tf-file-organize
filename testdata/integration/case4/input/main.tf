terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "aws" {
  region = "us-west-2"
}

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "subdomain" {
  description = "Subdomain for the service"
  type        = string
  default     = "api"
}

variable "domain_name" {
  description = "Domain name"
  type        = string
  default     = "example.com"
}

variable "build_number" {
  description = "Build number for container image"
  type        = string
  default     = "latest"
}

locals {
  environment = "production"
  common_tags = {
    Environment = local.environment
    Project     = "complex-test"
    ManagedBy   = "terraform"
  }
  service_name = "${var.subdomain}-service"
  full_domain  = "${var.subdomain}.${var.domain_name}"
}

# Google Cloud Run Service with complex nested blocks
resource "google_cloud_run_service" "complex_service" {
  name     = local.service_name
  location = var.region

  template {
    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = "10"
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

# AWS Security Group with complex ingress/egress rules
resource "aws_security_group" "complex_sg" {
  name_prefix = "${local.service_name}-"
  description = "Complex security group for ${local.service_name}"
  vpc_id      = aws_vpc.main.id

  # Multiple ingress rules with different configurations
  ingress {
    description = "HTTP from anywhere"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description     = "HTTPS from anywhere"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    cidr_blocks     = ["0.0.0.0/0"]
    prefix_list_ids = ["pl-12345678"]
  }

  ingress {
    description              = "SSH from management"
    from_port                = 22
    to_port                  = 22
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.management.id
  }

  ingress {
    description = "Custom app port"
    from_port   = 8080
    to_port     = 8090
    protocol    = "tcp"
    cidr_blocks = [
      "10.0.0.0/8",
      "172.16.0.0/12",
      "192.168.0.0/16"
    ]
  }

  # Multiple egress rules
  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description              = "Database access"
    from_port                = 5432
    to_port                  = 5432
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.database.id
  }

  dynamic "ingress" {
    for_each = var.additional_ports
    content {
      description = "Dynamic port ${ingress.value}"
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["10.0.0.0/8"]
    }
  }

  tags = merge(local.common_tags, {
    Name = "${local.service_name}-security-group"
    Type = "application"
  })
}

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

# Service account with complex configuration
resource "google_service_account" "service_account" {
  account_id   = "${local.service_name}-sa"
  display_name = "Service Account for ${local.service_name}"
  description  = "Service account for Cloud Run service ${local.service_name}"
}

resource "google_service_account" "monitoring" {
  account_id   = "${local.service_name}-monitoring"
  display_name = "Monitoring Service Account"
}

# SQL Database instance with complex configuration
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

# Secret Manager secret
resource "google_secret_manager_secret" "app_secret" {
  secret_id = "${local.service_name}-secret"

  replication {
    user_managed {
      replicas {
        location = var.region
        customer_managed_encryption {
          kms_key_name = google_kms_crypto_key.secret_key.id
        }
      }
      replicas {
        location = "us-east1"
      }
    }
  }

  labels = local.common_tags
}

# VPC and networking
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(local.common_tags, {
    Name = "${local.service_name}-vpc"
  })
}

resource "google_compute_network" "main" {
  name                    = "${local.service_name}-network"
  auto_create_subnetworks = false
  mtu                     = 1460
}

# KMS Key
resource "google_kms_crypto_key" "secret_key" {
  name     = "${local.service_name}-secret-key"
  key_ring = google_kms_key_ring.main.id

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = "SOFTWARE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_kms_key_ring" "main" {
  name     = "${local.service_name}-keyring"
  location = var.region
}

# Additional resources for dynamic blocks testing
resource "aws_security_group" "management" {
  name_prefix = "management-"
  vpc_id      = aws_vpc.main.id
  tags        = local.common_tags
}

resource "aws_security_group" "database" {
  name_prefix = "database-"
  vpc_id      = aws_vpc.main.id
  tags        = local.common_tags
}

# Outputs with complex expressions
output "service_url" {
  description = "URL of the Cloud Run service"
  value       = "https://${local.full_domain}"
}

output "service_internal_url" {
  description = "Internal URL of the service"
  value       = "https://${google_cloud_run_service.complex_service.status[0].url}"
}

output "database_connection" {
  description = "Database connection string"
  value       = "postgresql://user:password@${google_sql_database_instance.main.private_ip_address}:5432/database"
  sensitive   = true
}