variable "build_number" {
  description = "Build number for container image"
  type        = string
  default     = "latest"
}
variable "domain_name" {
  description = "Domain name"
  type        = string
  default     = "example.com"
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