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