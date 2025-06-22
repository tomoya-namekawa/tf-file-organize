resource "google_compute_network" "main" {
  auto_create_subnetworks = false
  mtu                     = 1460
  name                    = "${local.service_name}-network"
}
