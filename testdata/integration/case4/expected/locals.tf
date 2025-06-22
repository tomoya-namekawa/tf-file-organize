locals {
  common_tags  = {}
  environment  = "production"
  full_domain  = "${var.subdomain}.${var.domain_name}"
  service_name = "${var.subdomain}-service"
}
