resource "google_compute_network" "main" {
  name                    = "${local.service_name}-network"
  auto_create_subnetworks = false
  mtu                     = 1460
}