variable "build_number" {
  default     = "latest"
  description = "Build number for container image"
  type        = string
}

variable "domain_name" {
  default     = "example.com"
  description = "Domain name"
  type        = string
}

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  default     = "us-central1"
  description = "GCP region"
  type        = string
}

variable "subdomain" {
  default     = "api"
  description = "Subdomain for the service"
  type        = string
}
